package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"math"
	"net"
	"strings"

	"github.com/fxtlabs/primes"
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

type IsPrimeRequest struct {
	Method *string  `json:"method"`
	Number *float64 `json:"number"`
}

type IsPrimeResponse struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func handleConnect(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	invalidResponse := IsPrimeResponse{Method: "invalidResponse", Prime: false}
	invalidResponseByte, err := json.Marshal(invalidResponse)

	if err != nil {
		log.Fatal("Error marshalling JSON:", err)
		return
	}

	for {

		jsonString, err := reader.ReadString('\n')

		if err != nil {

			if err == io.EOF {
				log.Println("Connection closed")
				return
			}
			_, err := conn.Write(invalidResponseByte)

			if err != nil {
				log.Fatal(err)
			}

			return
		}

		validJsonString := strings.Replace(jsonString, "\n", "", 1)

		var data IsPrimeRequest

		err = json.Unmarshal([]byte(validJsonString), &data)

		if err != nil {
			_, err := conn.Write(invalidResponseByte)

			if err != nil {
				log.Fatal(err)
			}

			return
		}

		log.Println("Received data:", "method", data.Method, "number", data.Number)

		if data.Method == nil || *data.Method != "isPrime" || data.Number == nil {
			_, err := conn.Write(invalidResponseByte)

			if err != nil {
				log.Fatal(err)
			}

			return
		}

		var isPrime = *data.Number > 1 && checkIsPrime(*data.Number)

		response := IsPrimeResponse{
			Method: "isPrime",
			Prime:  isPrime,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Println("Error marshalling JSON:", err)
			return
		}

		jsonResponse = append(jsonResponse, byte('\n'))

		_, err = conn.Write(jsonResponse)
		if err != nil {
			log.Println("Error writing JSON response:", err)
			return
		}
	}

}

func checkIsPrime(number float64) bool {

	if number != math.Trunc(number) {
		return false
	}

	return primes.IsPrime(int(number))
}
