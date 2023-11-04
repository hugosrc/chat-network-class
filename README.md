type Message struct {
	Sender    string
	Receiver  string
	Content   string
	Timestamp time.Time
}

type User struct {
	Name     string
	Conn     net.Conn
	Messages chan Message
}

type Room struct {
	Name      string
	Users     map[string]User
	Message   []Message
	Join      chan User
	Leave     chan User
	Broadcast chan Message
}