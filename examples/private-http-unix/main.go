package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
)

func main() {
	socketPath := "/tmp/example.sock"

	// Clean up any old socket file
	if err := os.RemoveAll(socketPath); err != nil {
		fmt.Printf("Error removing old socket file: %v\n", err)
		return
	}

	// Create a Unix domain socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("Error creating Unix listener: %v\n", err)
		return
	}
	defer listener.Close() // Ensure the listener is closed when main exits

	fmt.Printf("HTTP server listening on Unix socket: %s\n", socketPath)

	// Define an HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from the Unix socket HTTP server!")
	})

	// Start the HTTP server using the Unix listener
	server := &http.Server{}
	if err := server.Serve(listener); err != nil {
		fmt.Printf("Error serving HTTP: %v\n", err)
	}
}
