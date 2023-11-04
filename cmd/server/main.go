package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hugosrc/chat/internal"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

	roomName := "Global"

	room := GetOrCreateRoom(roomName)
	room.JoinUser(userID, conn)

	defer room.LeaveUser(userID)

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}

		if messageType == websocket.TextMessage {
			message := string(p)
			fmt.Printf("Received message from %s: %s\n", userID, message)

			room.SendMessage(userID, message)
		}
	}
}

var rooms = make(map[string]*internal.Room)
var roomLock sync.RWMutex

func GetOrCreateRoom(roomName string) *internal.Room {
	roomLock.RLock()
	room, exists := rooms[roomName]
	roomLock.RUnlock()

	if exists {
		return room
	}

	roomLock.Lock()
	room = internal.NewRoom(roomName)
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
