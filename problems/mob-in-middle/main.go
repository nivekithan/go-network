package main

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
)

// Proxy connection are actual connection to the server
func proxyConnection(conn net.Conn) error {
	proxyConn, err := net.Dial("tcp", "chat.protohackers.com:16963")

	if err != nil {
		return err
	}

	proxyReader := bufio.NewReader(proxyConn)
	reader := bufio.NewReader(conn)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			msg, err := proxyReader.ReadString('\n')

			if err != nil {
				log.Println("Error while reading proxyReader msg")
				break
			}

			newMsg := rewriteBoguscoins(msg)

			if _, err := conn.Write([]byte(newMsg)); err != nil {
				break
			}
		}

		proxyConn.Close()
		conn.Close()

		log.Println("Clean shutdown of reading proxyConnection")
	}()

	go func() {
		defer wg.Done()
		for {
			msg, err := reader.ReadString('\n')

			if err != nil {
				break
			}

			newMsg := rewriteBoguscoins(msg)

			if _, err := proxyConn.Write([]byte(newMsg)); err != nil {
				break
			}

		}

		proxyConn.Close()
		conn.Close()
		log.Println("Clean shutdown of reading connection")
	}()

	wg.Wait()

	return nil
}

const tony = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"

func isAlnum(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z')
}

func rewriteBoguscoins(s string) string {
	var out strings.Builder
	out.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] == '7' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\n') {
			j := i
			for j < len(s) && isAlnum(s[j]) {
				j++
			}
			n := j - i
			if n >= 26 && n <= 35 && (j == len(s) || s[j] == ' ' || s[j] == '\n') {
				out.WriteString(tony)
				i = j
				continue
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

func handleConnection(conn net.Conn) {

	if err := proxyConnection(conn); err != nil {
		log.Printf("error: %v\n", err)
		return
	}
}

func run() error {
	lister, err := net.Listen("tcp", ":8000")
	if err != nil {
		return err
	}

	log.Println("Listening on port :8000")

	for {
		conn, err := lister.Accept()

		if err != nil {
			return err
		}

		go handleConnection(conn)
	}
}

func main() {

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
