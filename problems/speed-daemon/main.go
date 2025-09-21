package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
)

func clientError(conn net.Conn, msg string) error {
	clientError := ClientError{msg: msg}

	binary := clientError.toBinary()

	if _, err := conn.Write(binary); err != nil {
		return err
	}

	return errors.New(clientError.msg)
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
					return clientError(conn, fmt.Sprintf("Multiple heartbeats not allowed: %x", messageType))
				}

				isWantHeartbeat = true

				hearbeat, err := NewWantHeartbeat(reader)

				if err != nil {
					return err
				}

				go hearbeat.SendHeartBeat(conn)

			case 0x20:
				log.Println("MessageType=Plate")

				plate, err := NewPlate(reader)

				if err != nil {
					return err
				}

				log.Printf("%+v\n", plate)

			default:
				return clientError(conn, fmt.Sprintf("unknown messageType: %x", messageType))
			}
		}

	case 0x81:
		log.Println("MessageType=IamDispatcher")
		camera, err := NewIamDispatcher(reader)

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
					return clientError(conn, fmt.Sprintf("Multiple heartbeats not allowed: %x", messageType))
				}

				isWantHeartbeat = true

				hearbeat, err := NewWantHeartbeat(reader)

				if err != nil {
					return err
				}

				go hearbeat.SendHeartBeat(conn)

			default:
				return clientError(conn, fmt.Sprintf("unknown messageType: %x", messageType))
			}
		}

	default:
		return clientError(conn, fmt.Sprintf("unknown messageType: %x", messageType))
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
