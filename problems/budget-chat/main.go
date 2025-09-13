package main

import (
	"log"
	"net"
)

func run() error {
	port := ":8000"
	listen, err := net.Listen("tcp", port)

	if err != nil {
		return err
	}

	log.Printf("Listenting on port: %s\n", port)

	defer listen.Close()

	room := NewRoom()

	for {
		conn, err := listen.Accept()

		if err != nil {
			return err
		}

		go func() {
			user := NewUser(conn)

			if err := user.askAndSetName(); err != nil {
				log.Printf("error: %s", err)
				conn.Close()
				return
			}

			if err := room.addUser(user); err != nil {
				log.Printf("error: %s", err)
				conn.Close()
				return
			}
		}()
	}
}

func main() {

	if err := run(); err != nil {
		log.Fatal("error: ", err)
	}

}
