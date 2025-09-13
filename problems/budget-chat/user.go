package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
)

type User struct {
	mu     sync.Mutex
	id     string
	name   string
	conn   net.Conn
	reader bufio.Reader
}

func NewUser(conn net.Conn) *User {
	id := rand.Text()

	user := &User{
		conn:   conn,
		id:     id,
		reader: *bufio.NewReader(conn),
	}

	return user

}

func (u *User) askAndSetName() error {
	if err := u.write("Hey!, welcome to chat room. What's your name?"); err != nil {
		return fmt.Errorf("error writing message to user %s: %v", u.id, err)
	}

	name, err := u.read()

	if err != nil {
		return fmt.Errorf("error reading message from user %s: %v", u.id, err)
	}

	if !isValidName(name) {
		if err := u.write("Invalid name. Connect again with correct name"); err != nil {
			return fmt.Errorf("error writing message to user %s: %v", u.id, err)
		}
		return fmt.Errorf("Invalid name: %s", name)
	}

	log.Printf("Got valid name: %s\n", name)

	u.name = name

	return nil
}

func (u *User) close() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.conn.Close()
}

func (u *User) write(msg string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if _, err := fmt.Fprintf(u.conn, "%s\n", msg); err != nil {
		return err
	}

	return nil
}

func (u *User) read() (string, error) {
	msg, err := u.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	trimmedMessage := strings.TrimSpace(msg)

	return trimmedMessage, nil
}

func isValidName(name string) bool {
	if len(name) == 0 {
		return false
	}

	pattern := "^[0-9a-zA-Z]+$"

	re := regexp.MustCompile(pattern)

	result := re.MatchString(name)

	return result

}
