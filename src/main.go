package main

import (
	"io"
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

	byteCout, err := io.Copy(conn, conn)

	if err != nil {
		log.Println("Error writing data:", err)
		return
	}

	log.Println("Written data", "byteCount", byteCout)
}
