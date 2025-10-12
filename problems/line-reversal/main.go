package main

import (
	"bufio"
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
	reader := bufio.NewReader(conn)
	for {
		newLine, err := reader.ReadString('\n')

		if err != nil {
			log.Printf("Got error %v", err)
		}

		newLine = removeLastChar(newLine)

		log.Println(newLine)

		reversedNewLine := reverse(newLine)

		log.Printf("Reversed line: %s", reversedNewLine)

		output := []byte(reversedNewLine)

		output = append(output, '\n')

		conn.Write([]byte(output))

		log.Printf("Wrote line :%s\n", reversedNewLine)
	}

}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func removeLastChar(s string) string {
	if len(s) == 0 {
		return s
	}
	return s[:len(s)-1]
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
