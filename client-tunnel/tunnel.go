package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

/**
Taref (Tunnel Remote Forwarder) - is a package hel you to expose your local TCP through ssh tunnel provider

*/

type tunnel struct {
	host string
	port string
	auth *ssh.ClientConfig
}

type tcp struct {
	host string
	port string
}

type TunnelForwarder struct {
	tunnel   *tunnel
	listener *tcp
	service  *tcp
	zap      *zap.Logger
}

func NewTunnelRemoteForwarder(opts ...TunnelForwarderOpt) *TunnelForwarder {
	// init zap log
	logger, error := zap.NewProduction()
	if error != nil {
		log.Fatalln("[Taref] - Failed to init zap logger!")
	}
	defer logger.Sync()

	newTunnel := TunnelForwarder{
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
		zap: logger,
	}

	// set user configuration
	for _, opt := range opts {
		opt(&newTunnel)
	}

	return &newTunnel
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

func (tf *TunnelForwarder) ListenAndServe() error {
	if tf.listener == nil {
		errMsg := "No listerner host and port set"
		tf.zap.Error(errMsg)
		return errors.New(errMsg)
	}

	if tf.service == nil {
		errMsg := "No service host and port set"
		tf.zap.Error(errMsg)
		return errors.New(errMsg)
	}

	// Establish SSH connection
	sshClient, err := ssh.Dial("tcp", tf.getTunnelAddres(), tf.getTunnelAuth())
	if err != nil {
		tf.zap.Error("Failed to dial SSH server",
			zap.Error(err),
		)
		return err
	}
	defer sshClient.Close()

	tf.zap.Info(fmt.Sprintf("SSH connection established to %s", tf.getListenerAddres()))

	// Listen on the remote server
	listener, err := sshClient.Listen("tcp", tf.getListenerAddres())
	if err != nil {
		tf.zap.Error("Failed to listen on remote server",
			zap.Error(err),
		)
		return err
	}
	defer listener.Close()

	tf.zap.Info(fmt.Sprintf("Listening on remote server at %s. Forwarding to local service at %s", tf.getListenerAddres(), tf.getServiceAddres()))

	for {
		// Accept incoming connections on the remote listener
		remoteConn, err := listener.Accept()
		if err != nil {
			tf.zap.Error("Failed to accept remote connection",
				zap.Error(err),
			)
			continue
		}

		go func() {
			defer remoteConn.Close()

			tf.zap.Info(fmt.Sprintf("Accepted remote connection from %s", remoteConn.RemoteAddr()))

			// Dial the local service
			localConn, err := net.Dial("tcp", tf.getServiceAddres())
			if err != nil {
				tf.zap.Error("Failed to dial service",
					zap.Error(err),
				)
				return
			}
			defer localConn.Close()

			tf.zap.Info(fmt.Sprintf("Connected to service at %s", tf.getServiceAddres()))

			// Copy data between remote and local connections
			done := make(chan struct{})
			go func() {
				io.Copy(remoteConn, localConn)
				close(done)
			}()
			io.Copy(localConn, remoteConn)
			<-done // Wait for the other copy to finish

			tf.zap.Info(fmt.Sprintf("Connection closed for remote %s", remoteConn.RemoteAddr()))
		}()
	}
}
