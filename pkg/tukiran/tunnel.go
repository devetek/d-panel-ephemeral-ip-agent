package tukiran

/** Tukiran (Tunneling Unified Key Integration and Routing Access Network).

Is a package help you to expose your local TCP through ssh tunnel from any tunnel provider.

Copyright (c) 2025 Devetek. All rights reserved.
*/

import (
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"

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
	useSocket bool
	id        string
	tunnel    *tunnel
	listener  *tcp
	service   *tcp
	zap       *zap.Logger
	sshClient *ssh.Client
	state     ConnectionState
}

func NewTunnelRemoteForwarder(opts ...TunnelForwarderOpt) *TunnelForwarder {
	newTunnel := TunnelForwarder{
		tunnel:   new(tunnel),
		listener: new(tcp),
		service:  new(tcp),
	}

	// set user configuration
	for _, opt := range opts {
		opt(&newTunnel)
	}

	return &newTunnel
}

func (tf *TunnelForwarder) logger() *zap.Logger {
	if tf.zap == nil {
		return zap.NewNop()
	}

	return tf.zap.With(zap.Dict("module", zap.String("name", "tukiran")))
}

// set connection state
func (tf *TunnelForwarder) setState(state int) {
	tf.state = ConnectionState(state)
}

// get connection state
func (tf *TunnelForwarder) GetState() ConnectionState {
	return tf.state
}

// get connection state in human readable format
func (tf *TunnelForwarder) GetStateString() string {
	switch tf.state {
	case Idle:
		return "Idle"
	case Connecting:
		return "Connecting"
	case Connected:
		return "Connected"
	case Closed:
		return "Closed"
	case Error:
		return "Error"
	default:
		return "Unknown"
	}
}

// get protocol used
func (tf *TunnelForwarder) getUnixOrTCP() string {
	var n = "tcp"
	if tf.useSocket {
		n = "unix"
	}

	return n
}

// get connection id
func (tf *TunnelForwarder) GetID() string {
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
	if strings.HasPrefix(tf.listener.host, "/") {
		joinedPath := filepath.Join(tf.listener.host, tf.GetID())

		return joinedPath
	}

	return tf.listener.host + ":" + tf.listener.port
}

// set service address in tunnel client
func (tf *TunnelForwarder) getServiceAddres() string {
	return tf.service.host + ":" + tf.service.port
}

// get status connection
func (tf *TunnelForwarder) IsClosed() bool {
	return tf.GetState() == ConnectionState(3)
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
	listener, err := tf.sshClient.Listen(tf.getUnixOrTCP(), tf.getListenerAddres())
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
