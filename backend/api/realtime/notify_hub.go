package realtime

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

type Notification struct {
	ID        string    `json:"_id"`
	Details   string    `json:"details"`
	MainUID   string    `json:"mainuid"`
	TargetID  string    `json:"targetid"`
	IsReaded  bool      `json:"isreded"`
	CraetedAt time.Time `json:"createdAt"`
	User      User      `json:"user"`
}

type User struct {
	Name     string `json:"name"`
	ImageUrl string `json:"imageUrl"`
}

type NotificationManager struct {
	connections map[string]*websocket.Conn
	lock        sync.RWMutex
	kafkaMgr    *KafkaNotificationBridge
}

var notificationManager *NotificationManager

func InitNotificationManger(kafkaAddr, nodeID string) error {
	bridge, err := NewKafkaNotificationBridge(kafkaAddr, nodeID)
	if err != nil {
		return err
	}

	notificationManager = &NotificationManager{
		connections: make(map[string]*websocket.Conn),
		kafkaMgr:    bridge,
	}

	bridge.SetDeliveryHandler(notificationManager)
	return nil
}

func GetNotificationManager() *NotificationManager {
	return notificationManager
}

func (nm *NotificationManager) AddNotificationConnection(userID string, conn *websocket.Conn) {
	nm.lock.Lock()
	defer nm.lock.Unlock()

	// close old conn
	if oldConn, exists := nm.connections[userID]; exists {
		oldConn.Close()
	}
	nm.connections[userID] = conn
	log.Printf("User %s connected to notitificaton server", userID)
}

func (nm *NotificationManager) RemoveNotificationConnection(userID string) {
	nm.lock.Lock()
	defer nm.lock.Unlock()

	delete(nm.connections, userID)
	log.Printf("User %s disconnected from notitificaton server", userID)
}

func (nm *NotificationManager) SendNotificatonToUser(userID string, notifiaton Notification) error {
	return nm.kafkaMgr.PublishNotification(userID, notifiaton)
}

func (nm *NotificationManager) DeliverToLocalClient(userID string, notification Notification) {
	nm.lock.RLock()
	conn, exists := nm.connections[userID]
	nm.lock.RUnlock()

	if !exists {
		log.Printf("User %s not connected to this node", userID)
		return
	}

	err := conn.WriteJSON(notification)
	if err != nil {
		log.Printf("Error sending notifcation to user %s: %v", userID, err)
		nm.RemoveNotificationConnection(userID)
		return
	}
	log.Printf("Notification deliverd to user %s : %s", userID, notification.Details)
}

func (nm *NotificationManager) GetConnectedUsers() []string {
	nm.lock.RLock()
	defer nm.lock.RUnlock()

	users := make([]string, 0, len(nm.connections))
	for userID := range nm.connections {
		users = append(users, userID)
	}
	return users
}

func (nm *NotificationManager) Close() {
	if nm.kafkaMgr != nil {
		nm.kafkaMgr.Close()
	}
}
