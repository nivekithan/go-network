package main

import (
	"log"

	"github.com/nivekithan/go-network/problems/line-reversal/protocol"
)

func run() error {
	lis, err := protocol.NewListener(":8000")

	if err != nil {
		return err
	}

	defer lis.Close()

	log.Println("Listening for line reversal portocol at :8000")

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn *protocol.LineReversalConnection) {
	for {
		var newData [1000]byte

		n, err := conn.Read(newData[:])

		data := newData[:n]
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Got data: %s\n", data)
	}

}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
