package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

type Room struct {
	mu    sync.Mutex
	users map[string]*User
}

func NewRoom() *Room {
	return &Room{
		users: make(map[string]*User),
	}
}

func (r *Room) addUser(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[user.id]; ok {
		return fmt.Errorf("user with id: %s already present", user.id)
	}

	otherUsers := []string{}

	for _, otherUser := range r.users {
		otherUsers = append(otherUsers, otherUser.name)
	}

	r.broadcast(fmt.Sprintf("* %s has joined the room", user.name))

	r.users[user.id] = user

	go func(user *User) {
		defer func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			user.close()
			delete(r.users, user.id)
			log.Printf("user %s left the room\n", user.name)
			r.broadcast(fmt.Sprintf("* %s has left the room", user.name))
		}()

		currentUsersInfoMsg := fmt.Sprintf("* Current users: %s", strings.Join(otherUsers, ", "))

		if err := user.write(currentUsersInfoMsg); err != nil {
			return
		}

		for {
			msg, err := user.read()

			if err != nil {
				return
			}

			log.Printf("Got msg: %s", msg)

			r.broadcastExcept(fmt.Sprintf("[%s] %s", user.name, msg), user.id)
		}
	}(user)

	return nil
}

// Expects to be called by a function with lock hold
func (r *Room) broadcast(msg string) {
	for _, user := range r.users {
		if err := user.write(msg); err != nil {
			// This should never happen normally
			panic(err)
		}
	}
}

func (r *Room) broadcastExcept(msg string, userId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, user := range r.users {
		if user.id == userId {
			continue
		}

		if err := user.write(msg); err != nil {
			// This should never happen normally
			log.Printf("error: %v", err)
			return
		}
	}
}
