package realtime

import (
	"Server/kafka"
	"sync"

	"github.com/gofiber/websocket/v2"
)

type Message struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`
}

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan interface{}
	mu     sync.Mutex
	closed bool
}

type ChatHub struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	nodeID         string
	messageManager *kafka.MessageManager
	statusManager  *kafka.StatusManager
	getUserFriends func(string) <-chan []string
}
