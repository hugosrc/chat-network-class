package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type User struct {
	ID   string
	Conn *websocket.Conn
}

type Message struct {
	UserID  string `json:"userId"`
	Content string `json:"content"`
}

type Room struct {
	Name   string
	Users  map[string]*User
	Ch     chan Message
	Lock   sync.RWMutex
	Active bool
}

func NewRoom(name string) *Room {
	return &Room{
		Name:   name,
		Users:  make(map[string]*User),
		Ch:     make(chan Message),
		Lock:   sync.RWMutex{},
		Active: true,
	}
}

func (r *Room) BroadcastMessages() {
	for msg := range r.Ch {
		r.Lock.RLock()
		for _, user := range r.Users {
			if user.Conn != nil {
				if err := user.Conn.WriteJSON(msg); err != nil {
					fmt.Println("Error writing message:", err)
				}
			}
		}
		r.Lock.RUnlock()
	}
}

func (r *Room) JoinUser(userID string, conn *websocket.Conn) {
	user := &User{ID: userID, Conn: conn}
	r.Lock.Lock()
	r.Users[userID] = user
	r.Lock.Unlock()

	// Send a system message to inform everyone that a user has joined
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

	// Send a system message to inform everyone that a user has left
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

func handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Client connected")

	var userID string

	// Wait for the username message from the client
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading username message:", err)
			return
		}

		if messageType == websocket.TextMessage {
			message := string(p)
			var usernameMessage struct {
				Type     string `json:"type"`
				Username string `json:"username"`
			}

			if err := json.Unmarshal([]byte(message), &usernameMessage); err == nil {
				if usernameMessage.Type == "username" {
					userID = usernameMessage.Username
					break
				}
			}
		}
	}

	roomName := "General" // Specify the room name as needed

	room := GetOrCreateRoom(roomName)
	room.JoinUser(userID, conn)

	defer room.LeaveUser(userID)

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}

		// Assuming text message type (you can handle binary messages differently)
		if messageType == websocket.TextMessage {
			message := string(p)
			fmt.Printf("Received message from %s: %s\n", userID, message)

			room.SendMessage(userID, message)
		}
	}
}

var rooms = make(map[string]*Room)
var roomLock sync.RWMutex

func GetOrCreateRoom(roomName string) *Room {
	roomLock.RLock()
	room, exists := rooms[roomName]
	roomLock.RUnlock()

	if exists {
		return room
	}

	roomLock.Lock()
	room = NewRoom(roomName)
	rooms[roomName] = room
	go room.BroadcastMessages()
	roomLock.Unlock()

	return room
}

func main() {
	http.HandleFunc("/", handleConnection)
	fmt.Println("WebSocket server is running on :8080")
	http.ListenAndServe(":8080", nil)
}
