package main

import "log"

// create a client tunnel to connect to the server to replace native command ssh -N -R 2221:localhost:2222 -p 2220 tunne.dnocs.io
func main() {
	log.Println("Client tunnel started")
}
