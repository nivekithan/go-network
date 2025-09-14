package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

func run() error {

	packetConn, err := net.ListenPacket("udp", ":8000")

	if err != nil {
		return err
	}
	defer packetConn.Close()

	keyValueStore := make(map[string]string)

	log.Println("Listening for udp packets in :8000")

	for {
		var packet [1000]byte

		n, returnAddr, err := packetConn.ReadFrom(packet[:])

		if err != nil {
			return err
		}

		msg := string(packet[:n])

		log.Printf("Got msg: %s\n", msg)

		if strings.Contains(msg, "=") {
			// It is insert packet
			splittedStrings := strings.SplitN(msg, "=", 2)

			var key string
			var value string

			if len(splittedStrings) == 0 {
				key = ""
				value = ""
			} else if len(splittedStrings) == 1 {
				key = splittedStrings[0]
				value = ""
			} else {
				key = splittedStrings[0]
				value = splittedStrings[1]
			}

			if key == "version" {
				// You cannot update or insert a value to version key
				continue
			}

			log.Printf("key=%s\nvalue=%s", key, value)

			keyValueStore[key] = value
			continue
		}

		if msg == "version" {
			finalResponse := "version=1.0"
			if _, err := packetConn.WriteTo([]byte(finalResponse), returnAddr); err != nil {
				return err
			}
			continue
		}

		key := msg
		value := keyValueStore[key]

		finalResponse := fmt.Sprintf("%s=%s", key, value)

		log.Printf("key=%s\nvalue=%s", key, value)

		if _, err := packetConn.WriteTo([]byte(finalResponse), returnAddr); err != nil {
			return err
		}
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}

}
