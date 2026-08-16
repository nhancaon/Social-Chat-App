package realtime

import (
	"Server/kafka"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

type NotificationHub struct {
	connections         map[string]*websocket.Conn
	lock                sync.RWMutex
	notificationManager *kafka.NotificationManager
}

var notificationHubInstance *NotificationHub

func InitNotificationHub(kafkaAddr, nodeID string) error {
	notificationHub := &NotificationHub{
		connections: make(map[string]*websocket.Conn),
	}

	notificationManager, err := kafka.NewNotificationManager(kafkaAddr, nodeID, notificationHub)
	if err != nil {
		return err
	}
	notificationHub.notificationManager = notificationManager

	notificationHubInstance = notificationHub
	return nil
}

func GetNotificationHub() *NotificationHub {
	return notificationHubInstance
}

func (nh *NotificationHub) AddNotificationConnection(userID string, conn *websocket.Conn) {
	nh.lock.Lock()
	defer nh.lock.Unlock()

	// close old conn
	if oldConn, exists := nh.connections[userID]; exists {
		oldConn.Close()
	}
	nh.connections[userID] = conn
	log.Printf("User %s connected to notitificaton server", userID)
}

func (nh *NotificationHub) RemoveNotificationConnection(userID string) {
	nh.lock.Lock()
	defer nh.lock.Unlock()

	delete(nh.connections, userID)
	log.Printf("User %s disconnected from notitificaton server", userID)
}

func (nh *NotificationHub) SendNotificatonToUser(userID string, notifiaton kafka.Notification) error {
	return nh.notificationManager.PublishNotification(userID, notifiaton)
}

func (nh *NotificationHub) DeliverToLocalClient(userID string, notification kafka.Notification) {
	nh.lock.RLock()
	conn, exists := nh.connections[userID]
	nh.lock.RUnlock()

	if !exists {
		log.Printf("User %s not connected to this node", userID)
		return
	}

	err := conn.WriteJSON(notification)
	if err != nil {
		log.Printf("Error sending notifcation to user %s: %v", userID, err)
		nh.RemoveNotificationConnection(userID)
		return
	}
	log.Printf("Notification deliverd to user %s : %s", userID, notification.Details)
}

func (nh *NotificationHub) GetConnectedUsers() []string {
	nh.lock.RLock()
	defer nh.lock.RUnlock()

	users := make([]string, 0, len(nh.connections))
	for userID := range nh.connections {
		users = append(users, userID)
	}
	return users
}

func (nh *NotificationHub) Close() {
	if nh.notificationManager != nil {
		nh.notificationManager.Close()
	}
}
