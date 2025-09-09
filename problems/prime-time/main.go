package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8000")
	log.Println("Listening on port", "port", ":8000")

	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Fatal(err)
		}

		go handleConnect(conn)
	}
}

func handleConnect(conn net.Conn) {
	defer conn.Close()

	// TODO: Implement prime-time protocol
	// Prime-time is a JSON-based protocol for testing primality
	log.Println("Connection received - prime-time implementation needed")
}
