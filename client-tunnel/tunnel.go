package main

/** Tukiran (Tunneling Unified Key Integration and Routing Access Network).

Is a package help you to expose your local TCP through ssh tunnel from any tunnel provider.
Usualy when you want to connect to SSH tunnel, you need to run command `ssh -N -L <local-port>:<remote-host>:<remote-port> <tunnel-server>`.

Copyright (c) 2025 Devetek. All rights reserved.
*/

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type tunnel struct {
	host string
	port string
	auth *ssh.ClientConfig
}

type tcp struct {
	host string
	port string
}

type ConnectionState int

const (
	Idle ConnectionState = iota
	Connecting
	Connected
	Closed
	Error
)

type TunnelForwarder struct {
	id        string
	tunnel    *tunnel
	listener  *tcp
	service   *tcp
	zap       *zap.Logger
	sshClient *ssh.Client
	state     ConnectionState
	closed    bool
}

func NewTunnelRemoteForwarder(opts ...TunnelForwarderOpt) *TunnelForwarder {
	logger, error := zap.NewProduction()
	if error != nil {
		logger.Fatal(
			"Failed to init zap logger!",
			zap.Error(error),
			zap.Dict("module", zap.String("name", "tukiran")),
		)
	}
	defer logger.Sync()

	newTunnel := TunnelForwarder{
		id: time.Now().String(),
		tunnel: &tunnel{
			host: "localhost",
			port: "22",
			auth: &ssh.ClientConfig{},
		},
		listener: &tcp{
			host: "localhost",
			port: "80",
		},
		service: &tcp{
			host: "localhost",
			port: "3000",
		},
		zap:    logger,
		closed: false,
	}

	// set user configuration
	for _, opt := range opts {
		opt(&newTunnel)
	}

	return &newTunnel
}

func (tf *TunnelForwarder) logger() *zap.Logger {
	return tf.zap.With(zap.Dict("module", zap.String("name", "tukiran")))
}

// set connection state
func (tf *TunnelForwarder) setState(state int) {
	tf.state = ConnectionState(state)
}

// get connection state
func (tf *TunnelForwarder) getState() ConnectionState {
	return tf.state
}

// get connection id
func (tf *TunnelForwarder) getID() string {
	return tf.id
}

// get tunnel server address
func (tf *TunnelForwarder) getTunnelAddres() string {
	return tf.tunnel.host + ":" + tf.tunnel.port
}

// get tunnel server address
func (tf *TunnelForwarder) getTunnelAuth() *ssh.ClientConfig {
	return tf.tunnel.auth
}

// get listener address in tunnel server
func (tf *TunnelForwarder) getListenerAddres() string {
	return tf.listener.host + ":" + tf.listener.port
}

// set service address in tunnel client
func (tf *TunnelForwarder) getServiceAddres() string {
	return tf.service.host + ":" + tf.service.port
}

// get status connection
func (tf *TunnelForwarder) IsClosed() bool {
	if tf.getState() == ConnectionState(3) {
		return true
	}

	return false
}

func (tf *TunnelForwarder) ListenAndServe() error {
	if tf.listener == nil {
		errMsg := "No listerner host and port set"
		tf.logger().Error(errMsg)
		return errors.New(errMsg)
	}

	if tf.service == nil {
		errMsg := "No service host and port set"
		tf.logger().Error(errMsg)
		return errors.New(errMsg)
	}

	// Establish SSH connection
	var err error

	tf.sshClient, err = ssh.Dial("tcp", tf.getTunnelAddres(), tf.getTunnelAuth())
	if err != nil {
		tf.logger().Error("Failed to dial SSH server",
			zap.Error(err),
		)
		return err
	}
	defer tf.sshClient.Close()

	// set connected state
	tf.setState(2)

	tf.logger().Info(fmt.Sprintf("SSH connection established to %s", tf.getTunnelAddres()))

	// Listen on the remote server
	listener, err := tf.sshClient.Listen("tcp", tf.getListenerAddres())
	if err != nil {
		tf.setState(4)
		tf.logger().Error("Failed to listen on remote server",
			zap.Error(err),
		)
		return err
	}
	defer listener.Close()

	tf.logger().Info(fmt.Sprintf("Listening on remote server at %s. Forwarding to local service at %s", tf.getListenerAddres(), tf.getServiceAddres()))

	for {
		// Accept incoming connections on the remote listener
		remoteConn, err := listener.Accept()
		if err != nil {
			// check if connection already close, break loop
			if tf.IsClosed() {
				break
			}
			// set connection status to closed
			tf.setState(3)

			tf.logger().Error("Failed to accept remote connection",
				zap.Error(err),
			)
			continue
		}

		go func() {
			defer remoteConn.Close()

			tf.logger().Info(fmt.Sprintf("Accepted remote connection from %s", remoteConn.RemoteAddr()))

			// Dial the local service
			localConn, err := net.Dial("tcp", tf.getServiceAddres())
			if err != nil {
				tf.setState(4)
				tf.logger().Error("Failed to dial service",
					zap.Error(err),
				)
				return
			}
			defer localConn.Close()

			tf.logger().Info(fmt.Sprintf("Connected to service at %s", tf.getServiceAddres()))

			// Copy data between remote and local connections
			done := make(chan struct{})
			go func() {
				io.Copy(remoteConn, localConn)
				close(done)
			}()
			io.Copy(localConn, remoteConn)
			<-done // Wait for the other copy to finish

			tf.logger().Info(fmt.Sprintf("Connection closed for remote %s", remoteConn.RemoteAddr()))
		}()
	}

	return nil
}

func (tf *TunnelForwarder) Close() {
	if tf.sshClient != nil {
		tf.setState(3)
		tf.sshClient.Close()
	}
}
