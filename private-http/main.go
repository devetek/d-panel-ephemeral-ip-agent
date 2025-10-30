package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Define a handler function for the "/hello" route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, Go HTTP Server!")
	})

	// Start the HTTP server on port 8080
	fmt.Println("Server starting on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
