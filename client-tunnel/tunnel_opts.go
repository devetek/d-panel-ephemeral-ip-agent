package main

import "golang.org/x/crypto/ssh"

type TunnelForwarderOpt func(*TunnelForwarder)

// set tunnel host, you can use `tunnel.dnocs.io` or another tunnel providers
func WithTunnelHost(host string) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.tunnel.host = host
	}
}

// set tunnel port if not using default ssh port
func WithTunnelPort(port string) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.tunnel.port = port
	}
}

// set tunnel authentication method
func WithTunnelAuthMethod(authMethod *ssh.ClientConfig) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.tunnel.auth = authMethod
	}
}

// set listener host
func WithListenerHost(host string) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.listener.host = host
	}
}

// set listener port
func WithListenerPort(port string) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.listener.port = port
	}
}

// set service host
func WithServiceHost(host string) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.service.host = host
	}
}

// set service port
func WithServicePort(port string) func(*TunnelForwarder) {
	return func(tf *TunnelForwarder) {
		tf.service.port = port
	}
}
