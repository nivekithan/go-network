package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
)

func clientError(msg string) []byte {
	clientError := ClientError{msg: msg}

	return clientError.toBinary()

}

func hanldeConnectionImp(conn net.Conn) error {

	reader := bufio.NewReader(conn)

	var messageType uint8

	if err := binary.Read(reader, binary.BigEndian, &messageType); err != nil {
		return err
	}

	switch messageType {
	case 0x80:
		log.Println("MessageType=IamCamera")
		camera, err := NewIamCamera(reader)

		if err != nil {
			return err
		}

		log.Printf("%+v\n", camera)

		isWantHeartbeat := false

		for {
			var messageType uint8

			if err := binary.Read(reader, binary.BigEndian, &messageType); err != nil {
				return err
			}

			switch messageType {
			case 0x40:
				log.Println("MessageType=WantHeartbeat")

				if isWantHeartbeat {
					error := ClientError{msg: fmt.Sprintf("Multiple heartbeats not allowed: %x", messageType)}

					if _, err := conn.Write(error.toBinary()); err != nil {
						return err
					}
					return errors.New(error.msg)
				}

				isWantHeartbeat = true

				hearbeat, err := NewWantHeartbeat(reader)

				if err != nil {
					return err
				}

				go hearbeat.SendHeartBeat(conn)

			case 0x20:
				log.Println("MessageType=Plate")
			default:
				error := ClientError{msg: fmt.Sprintf("unknown messageType: %x", messageType)}

				if _, err := conn.Write(error.toBinary()); err != nil {
					return err
				}

				return errors.New(error.msg)
			}
		}

	case 0x81:
		log.Println("MessageType=IamDispatcher")
		camera, err := NewIamDispatcher(reader)

		if err != nil {
			return err
		}
		log.Printf("%+v\n", camera)

		for {

		}

	default:
		error := ClientError{msg: fmt.Sprintf("unknown messageType: %x", messageType)}

		if _, err := conn.Write(error.toBinary()); err != nil {
			return err
		}

		return errors.New(error.msg)
	}
}

// This function blocks
func handleConnection(conn net.Conn) {
	defer conn.Close()
	defer log.Println("Closing connection")

	if err := hanldeConnectionImp(conn); err != nil {
		log.Println("error: ", err)
	}
}

func handleListner(listner net.Listener) error {
	conn, err := listner.Accept()

	if err != nil {
		return err
	}

	go handleConnection(conn)

	return nil
}

func run() error {

	listner, err := net.Listen("tcp", ":8000")

	if err != nil {
		return err
	}

	log.Println("Listening in port :8000")

	for {

		err := handleListner(listner)

		if err != nil {
			log.Println("error: handleListner", err)
		}

	}

}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
