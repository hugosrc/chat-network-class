package internal

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

type Room struct {
	Name   string
	Users  map[string]*User
	Ch     chan Message
	Lock   sync.Mutex
	Active bool
}

func NewRoom(name string) *Room {
	return &Room{
		Name:   name,
		Users:  make(map[string]*User),
		Ch:     make(chan Message),
		Lock:   sync.Mutex{},
		Active: true,
	}
}

func (r *Room) BroadcastMessages() {
	for msg := range r.Ch {
		r.Lock.Lock()
		for _, user := range r.Users {
			if user.Conn != nil {
				if err := user.Conn.WriteJSON(msg); err != nil {
					fmt.Println("Error writing message:", err)
				}
			}
		}
		r.Lock.Unlock()
	}
}

func (r *Room) JoinUser(userID string, conn *websocket.Conn) {
	user := &User{ID: userID, Conn: conn}
	r.Lock.Lock()
	r.Users[userID] = user
	r.Lock.Unlock()

	systemMessage := Message{
		UserID:  "System",
		Content: fmt.Sprintf("%s entrou", userID),
	}
	r.Ch <- systemMessage
}

func (r *Room) LeaveUser(userID string) {
	r.Lock.Lock()
	delete(r.Users, userID)
	r.Lock.Unlock()

	systemMessage := Message{
		UserID:  "System",
		Content: fmt.Sprintf("%s saiu", userID),
	}
	r.Ch <- systemMessage
}

func (r *Room) SendMessage(userID, content string) {
	msg := Message{
		UserID:  userID,
		Content: content,
	}
	r.Ch <- msg
}
