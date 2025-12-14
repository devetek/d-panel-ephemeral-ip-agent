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
	"sync"
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
	useSocket bool
	id        string
	tunnel    *tunnel
	listener  *tcp
	service   *tcp
	zap       *zap.Logger
	sshClient *ssh.Client
	state     ConnectionState
	wg        sync.WaitGroup
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
func (tf *TunnelForwarder) setState(state ConnectionState) {
	tf.state = state
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

func (tf *TunnelForwarder) ListenAndServeHTTP() error {
	// set initial state to connecting
	tf.setState(Connecting)

	if tf.listener == nil {
		tf.setState(Error)
		errMsg := "no listerner host and port set"
		tf.logger().Error(errMsg)
		return errors.New(errMsg)
	}

	if tf.service == nil {
		tf.setState(Error)
		errMsg := "no service host and port set"
		tf.logger().Error(errMsg)
		return errors.New(errMsg)
	}

	// Establish SSH connection
	var err error

	tf.sshClient, err = ssh.Dial("tcp", tf.getTunnelAddres(), tf.getTunnelAuth())
	if err != nil {
		tf.setState(Error)
		tf.logger().Error("Failed to dial SSH server",
			zap.Error(err),
		)
		return err
	}
	defer tf.sshClient.Close()

	// send keepalive evry 10 seconds
	go func() {
		for {
			if tf.IsClosed() {
				break
			}
			_, _, err := tf.sshClient.SendRequest("", false, nil)
			if err != nil {
				tf.logger().Error("Failed to send keepalive request",
					zap.Error(err),
				)
				break
			}

			// wait for 10 seconds
			time.Sleep(10 * time.Second)
		}
	}()

	// set connection state to connected
	tf.setState(Connected)
	tf.logger().Info(fmt.Sprintf("SSH connection established to %s", tf.getTunnelAddres()))

	// Listen on the remote server
	listener, err := tf.sshClient.Listen(tf.getUnixOrTCP(), tf.getListenerAddres())
	if err != nil {
		tf.setState(Error)
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
				tf.logger().Info("Listener closed, stopping accept loop")
				break
			}
			// set connection status to closed
			tf.setState(Closed)

			tf.logger().Error("Failed to accept remote connection",
				zap.Error(err),
			)
			continue
		}

		go func() {
			defer remoteConn.Close()

			// Dial the local service
			localConn, err := net.DialTimeout("tcp", tf.getServiceAddres(), 10*time.Second)
			if err != nil {
				tf.setState(Error)
				tf.logger().Error("Failed to dial service",
					zap.Error(err),
				)
				return
			}
			defer localConn.Close()

			// wait 2 routines to finish
			tf.wg.Add(2)

			// Remote (client) -> Local (server)
			go func() {
				defer tf.wg.Done()
				// io.Copy blocks until EOF or error
				if _, err := io.Copy(localConn, remoteConn); err != nil && err != io.EOF {
					tf.logger().Error(fmt.Sprintf("remote to local copy error: %v", err))
				}
				// Crucial: After the remote client finishes sending data (EOF on remoteConn read side),
				// we can signal to our local server that the request body is complete.
				// This is a TCP half-closure equivalent.
				if tcpConn, ok := localConn.(*net.TCPConn); ok {
					tcpConn.CloseWrite() // Signal local HTTP server we are done writing
				}
			}()

			// Local (server) -> Remote (client)
			go func() {
				// close connections and mark done when finished for HTTP after connection copied
				defer localConn.Close()
				defer remoteConn.Close()
				defer tf.wg.Done()

				// io.Copy blocks until EOF or error from the local server response
				if _, err := io.Copy(remoteConn, localConn); err != nil && err != io.EOF {
					tf.logger().Error(fmt.Sprintf("local to remote copy error: %v", err))
				}

				// set state to closed after finished, to notify consumer if HTTP connection is closed
				if remoteConn == nil {
					tf.setState(Closed)
				}
			}()

			tf.wg.Wait() // Wait for both directions to fully complete
		}()
	}

	return nil
}

func (tf *TunnelForwarder) ListenAndServe() error {
	// set initial state to connecting
	tf.setState(Connecting)

	if tf.listener == nil {
		tf.setState(Error)
		errMsg := "no listerner host and port set"
		tf.logger().Error(errMsg)
		return errors.New(errMsg)
	}

	if tf.service == nil {
		tf.setState(Error)
		errMsg := "no service host and port set"
		tf.logger().Error(errMsg)
		return errors.New(errMsg)
	}

	// Establish SSH connection
	var err error

	tf.sshClient, err = ssh.Dial("tcp", tf.getTunnelAddres(), tf.getTunnelAuth())
	if err != nil {
		tf.setState(Error)
		tf.logger().Error("Failed to dial SSH server",
			zap.Error(err),
		)
		return err
	}
	defer tf.sshClient.Close()

	// send keepalive evry 10 seconds
	go func() {
		for {
			if tf.IsClosed() {
				break
			}
			_, _, err := tf.sshClient.SendRequest("", false, nil)
			if err != nil {
				tf.logger().Error("Failed to send keepalive request",
					zap.Error(err),
				)
				break
			}

			// wait for 10 seconds
			time.Sleep(10 * time.Second)
		}
	}()

	// set connection state to connected
	tf.setState(Connected)
	tf.logger().Info(fmt.Sprintf("SSH connection established to %s", tf.getTunnelAddres()))

	// Listen on the remote server
	listener, err := tf.sshClient.Listen(tf.getUnixOrTCP(), tf.getListenerAddres())
	if err != nil {
		tf.setState(Error)
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
				tf.logger().Info("Listener closed, stopping accept loop")
				break
			}
			// set connection status to closed
			tf.setState(Closed)

			tf.logger().Error("Failed to accept remote connection",
				zap.Error(err),
			)
			continue
		}

		go func() {
			defer remoteConn.Close()

			// Dial the local service
			localConn, err := net.DialTimeout("tcp", tf.getServiceAddres(), 10*time.Second)
			if err != nil {
				tf.setState(Error)
				tf.logger().Error("Failed to dial service",
					zap.Error(err),
				)
				return
			}
			defer localConn.Close()

			// Copy data between remote and local connections (currently works well for SSH)
			// Never close connection, let client or server close it
			done := make(chan struct{})
			go func() {
				io.Copy(remoteConn, localConn)
				close(done)
			}()
			io.Copy(localConn, remoteConn)

			<-done // Wait for the other copy to finish
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
