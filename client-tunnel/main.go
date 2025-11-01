package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	manager := NewManager(
		// WithSource(ConfigSourceRemote),
		// WithURL("https://raw.githubusercontent.com/dPanel-ID/version/refs/heads/main/tunnel-dev.json"),
		WithSource(ConfigSourceFile),
		WithURL("./client-tunnel/config.json"),
	)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Starting tunnel client")

	manager.Start()

	<-done

	log.Println("Stopping tunnel client")
}
