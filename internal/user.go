package internal

import "github.com/gorilla/websocket"

type User struct {
	ID   string
	Conn *websocket.Conn
}
