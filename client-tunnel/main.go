package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
)

func main() {
	tunnelSSH := NewTunnelRemoteForwarder(
		WithTunnelHost("tunnel.beta.devetek.app"),
		WithTunnelPort("2220"),
		WithTunnelAuthMethod(&ssh.ClientConfig{
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}),
		WithListenerHost("0.0.0.0"),
		WithListenerPort("2221"),
		WithServiceHost("localhost"),
		WithServicePort("2222"),
	)

	tunnelHTTP := NewTunnelRemoteForwarder(
		WithTunnelHost("tunnel.beta.devetek.app"),
		WithTunnelPort("2220"),
		WithTunnelAuthMethod(&ssh.ClientConfig{
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}),
		WithListenerHost("0.0.0.0"),
		WithListenerPort("3001"),
		WithServiceHost("localhost"),
		WithServicePort("3000"),
	)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Starting tunnel client")
	go func() {
		err := tunnelSSH.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()
	go func() {
		err := tunnelHTTP.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	<-done
	log.Println("Stopping tunnel client")
}
