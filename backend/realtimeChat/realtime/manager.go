package realtime

import (
	"log"
	"sync"

	"realTimeChat/gapi"

	"github.com/gofiber/websocket/v2"
)

type Message struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`
}

type ConnectionManager struct {
	connections    map[string]*websocket.Conn
	onlineFriends  map[string][]string
	getUserFriends func(string) <-chan []string
	lock           sync.Mutex
}

func NewConnectionManager(getUserFriends func(string) <-chan []string) *ConnectionManager {
	if getUserFriends == nil {
		return nil
	}
	return &ConnectionManager{
		connections:    make(map[string]*websocket.Conn),
		onlineFriends:  make(map[string][]string),
		getUserFriends: getUserFriends,
	}
}

// contains reports whether id is already present in list.
func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

// notification is a pending "onlineFriends" push to a single connection.
// Building these while the lock is held, then sending them after unlocking,
// keeps slow/broken sockets from blocking every other user.
type notification struct {
	conn *websocket.Conn
	id   string
	list []string
}

func (cm *ConnectionManager) AddConnection(userID string, conn *websocket.Conn) {
	// Fetch this user's friend list BEFORE taking the lock. getUserFriends
	// makes a blocking gRPC call; doing it while holding cm.lock would stall
	// every other connect/disconnect/message on the server until it returns.
	myFriends := <-cm.getUserFriends(userID)
	friendSet := make(map[string]bool, len(myFriends))
	for _, f := range myFriends {
		friendSet[f] = true
	}

	cm.lock.Lock()

	// If this user already had a connection open (e.g. reconnect, duplicate
	// tab), close the old socket so it doesn't leak.
	if oldConn, ok := cm.connections[userID]; ok && oldConn != nil {
		oldConn.Close()
	}

	cm.connections[userID] = conn
	cm.onlineFriends[userID] = []string{}

	var notifications []notification
	for friendID, friendConn := range cm.connections {
		if friendID == userID || friendConn == nil || !friendSet[friendID] {
			continue
		}
		// Mark each other online (assumes mutual friendship).
		if !contains(cm.onlineFriends[friendID], userID) {
			cm.onlineFriends[friendID] = append(cm.onlineFriends[friendID], userID)
		}
		if !contains(cm.onlineFriends[userID], friendID) {
			cm.onlineFriends[userID] = append(cm.onlineFriends[userID], friendID)
		}
		notifications = append(notifications, notification{
			conn: friendConn,
			id:   friendID,
			list: cm.onlineFriends[friendID],
		})
	}
	myList := cm.onlineFriends[userID]

	cm.lock.Unlock()

	// Do all network I/O after releasing the lock.
	for _, n := range notifications {
		if err := n.conn.WriteJSON(map[string]interface{}{"onlineFriends": n.list}); err != nil {
			log.Printf("Error notifying %s about %s: %v", n.id, userID, err)
		}
	}
	if err := conn.WriteJSON(map[string]interface{}{"onlineFriends": myList}); err != nil {
		log.Printf("Error notifying %s about their online friends: %v", userID, err)
	}
}

func (cm *ConnectionManager) RemoveConnection(userID string) {
	cm.lock.Lock()

	delete(cm.connections, userID)
	delete(cm.onlineFriends, userID)

	var notifications []notification
	for friendID, list := range cm.onlineFriends {
		for i, id := range list {
			if id == userID {
				cm.onlineFriends[friendID] = append(list[:i], list[i+1:]...)
				if conn, ok := cm.connections[friendID]; ok && conn != nil {
					notifications = append(notifications, notification{
						conn: conn,
						id:   friendID,
						list: cm.onlineFriends[friendID],
					})
				}
				break
			}
		}
	}

	cm.lock.Unlock()

	for _, n := range notifications {
		if err := n.conn.WriteJSON(map[string]interface{}{"onlineFriends": n.list}); err != nil {
			log.Printf("Error notifying %s about %s: %v", n.id, userID, err)
		}
	}
}

func (cm *ConnectionManager) SendToReceiver(msg Message) {
	cm.lock.Lock()
	conn, ok := cm.connections[msg.Receiver]
	cm.lock.Unlock()

	if !ok || conn == nil {
		log.Printf("Receiver %s not found", msg.Receiver)
		return
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending message to %s: %v", msg.Receiver, err)
	}

	// Save message to DB via gRPC. A failure here must NOT crash the server
	// (log.Fatalf would call os.Exit and kill every connected user).
	if err := gapi.SendMessageClient(msg.Sender, msg.Receiver, msg.Content); err != nil {
		log.Printf("Error saving message to gRPC for %s: %v", msg.Receiver, err)
	}
}
