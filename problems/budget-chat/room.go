package main

import (
	"fmt"
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

	go func() {
		defer func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			user.close()
			delete(r.users, user.id)
		}()

		user.listenForChatMessages(otherUsers)
	}()

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
