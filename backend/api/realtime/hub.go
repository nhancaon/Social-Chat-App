package realtime

import (
	"Server/database"
	"Server/kafka"
	"Server/models"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var chatHubInstance *ChatHub

// GetChatHub returns the process-wide ChatHub set up by InitChatHub, or nil
// if it hasn't been initialized yet.
func GetChatHub() *ChatHub {
	return chatHubInstance
}

// InitChatHub creates the process-wide ChatHub and makes it available via
// GetChatHub.
func InitChatHub(kafkaAddr, nodeID string) (*ChatHub, error) {
	chatHub, err := NewChatHub(kafkaAddr, nodeID)
	if err != nil {
		return nil, err
	}
	chatHubInstance = chatHub
	return chatHub, nil
}

func NewChatHub(kafkaAddr, nodeID string) (*ChatHub, error) {
	chatHub := &ChatHub{
		clients:        make(map[string]*Client),
		nodeID:         nodeID,
		getUserFriends: GetUserFriends,
	}

	statusManager, err := kafka.NewStatusManager(kafkaAddr, nodeID, chatHub)
	if err != nil {
		return nil, err
	}

	chatHub.statusManager = statusManager

	messageManager, err := kafka.NewMessageManager(kafkaAddr, nodeID, chatHub)
	if err != nil {
		statusManager.Close()
		return nil, err
	}
	chatHub.messageManager = messageManager

	log.Printf("%s Chat hub ready", nodeID)
	return chatHub, nil
}

func (h *ChatHub) RegisterClient(userID string, conn *websocket.Conn) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan interface{}, 256),
		closed: false,
	}

	h.clients[userID] = client

	status := kafka.UserStatus{
		UserID:    userID,
		NodeID:    h.nodeID,
		Online:    true,
		Timestamp: time.Now().Unix(),
	}

	go func() {
		if err := h.statusManager.PublishStatus(status); err != nil {
			log.Printf("Faild to publish online status for user %s, %v", userID, err)
		}
	}()

	log.Printf("User %s connected on node %s", userID, h.nodeID)

	// send online
	go h.sendOnlineFriendsToUser(userID)
	return client
}

func (h *ChatHub) sendOnlineFriendsToUser(userID string) {
	friendsChan := h.getUserFriends(userID)
	friends := <-friendsChan

	if friends == nil {
		return
	}

	onlineUsers := h.GetOnlineUsers()
	onlineFriends := []string{}

	for _, friendID := range friends {
		if _, isOnline := onlineUsers[friendID]; isOnline {
			onlineFriends = append(onlineFriends, friendID)
		}
	}

	h.mu.RLock()
	client, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if !client.closed {
		select {
		case client.Send <- map[string]interface{}{
			"onlineFriends": onlineFriends,
		}:
			log.Printf("Send %d online friends to user %s", len(onlineFriends), userID)
		default:
			log.Printf("Could not send online frids to user %s (channel full)", userID)
		}
	}

}

func (h *ChatHub) UnregisterClient(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.mu.Lock()
		if !client.closed {
			close(client.Send)
			client.closed = true
		}
		client.mu.Unlock()

		delete(h.clients, userID)

		status := kafka.UserStatus{
			UserID:    userID,
			NodeID:    h.nodeID,
			Online:    false,
			Timestamp: time.Now().Unix(),
		}

		go func() {
			if err := h.statusManager.PublishStatus(status); err != nil {
				log.Printf("Faidl to publish status for user %s :%v", userID, err)
			}
		}()

		log.Printf("User %s disconnected form node %s", userID, h.nodeID)
	}
}

// HandlerUserStatus
func (h *ChatHub) HandlerUserStatus(status kafka.UserStatus) {
	log.Printf("User %s status : online=%v (node : %s)", status.UserID, status.Online, status.NodeID)
	h.notifyFriendsAboutStatusChange(status)
}

func (h *ChatHub) notifyFriendsAboutStatusChange(status kafka.UserStatus) {
	friendsChan := h.getUserFriends(status.UserID)
	friends := <-friendsChan

	if friends == nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, friendID := range friends {
		if client, ok := h.clients[friendID]; ok {
			go func(fID string, c *Client) {
				friendsChan := h.getUserFriends(fID)
				friendsList := <-friendsChan

				if friendsList == nil {
					return
				}

				onlineUsers := h.GetOnlineUsers()
				onlineFriends := []string{}

				for _, f := range friendsList {
					if _, isOnline := onlineUsers[f]; isOnline {
						onlineFriends = append(onlineFriends, f)
					}
				}

				c.mu.Lock()
				defer c.mu.Unlock()

				if !c.closed {
					select {
					case c.Send <- map[string]interface{}{
						"onlineFriends": onlineFriends,
					}:
						log.Printf("Updated online friends for user %s", fID)
					default:
						log.Printf("Could not update online friends for user %s", fID)
					}
				}

			}(friendID, client)
		}
	}

}

// DeliverMessage
func (h *ChatHub) DeliverMessage(msg *kafka.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	response := map[string]interface{}{
		"sender":   msg.FromUserID,
		"receiver": msg.ToUserID,
		"content":  msg.Content,
	}

	if recipient, ok := h.clients[msg.ToUserID]; ok {
		recipient.mu.Lock()
		if !recipient.closed {
			select {
			case recipient.Send <- response:
				log.Printf("Message Deliverd to %s", msg.ToUserID)
			default:
				log.Printf("Send Channel full for user %s", msg.ToUserID)
			}
		}
		recipient.mu.Unlock()
	}
}

func (h *ChatHub) SendMessageWithRetry(msg *kafka.Message, maxRetries int) error {
	// save to db
	if err := h.saveMessageToDB(msg); err != nil {
		log.Printf("Faild to save message to db :%v", err)
		return err
	}

	return h.messageManager.SendMessageWithRetry(msg, maxRetries)
}

// saveMessageToDB DataBase Operations func

func (h *ChatHub) saveMessageToDB(msg *kafka.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if msg.FromUserID == "" || msg.ToUserID == "" {
		return fmt.Errorf("sender and reciver are required")
	}

	UserSchema := database.DB.Collection("users")

	// check s & r are exists
	senderID, _ := primitive.ObjectIDFromHex(msg.FromUserID)
	receiverID, _ := primitive.ObjectIDFromHex(msg.ToUserID)

	var sender, receiver models.UserModel
	err := UserSchema.FindOne(ctx, bson.M{"_id": senderID}).Decode(&sender)
	if err != nil {
		return fmt.Errorf("sender not found")
	}
	err = UserSchema.FindOne(ctx, bson.M{"_id": receiverID}).Decode(&receiver)
	if err != nil {
		return fmt.Errorf("receiver not found")
	}

	// save msg
	message := models.Message{
		Content:   msg.Content,
		Sender:    msg.FromUserID,
		Receiver:  msg.ToUserID,
		CreatedAt: time.Now(),
	}

	_, err = database.DB.Collection("messages").InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("faild to save message to db")
	}

	// update unread message count for the receiver
	unreadMsgSchema := database.DB.Collection("unReadmessages")

	filter := bson.M{"mainUserid": msg.ToUserID, "otherUserid": msg.FromUserID}
	update := bson.M{"$inc": bson.M{"numOfUnreadMessages": 1}, "$set": bson.M{"isReadMessage": false}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var unreadMsg models.UnReadMsg
	if err := unreadMsgSchema.FindOneAndUpdate(ctx, filter, update, opts).Decode(&unreadMsg); err != nil {
		return fmt.Errorf("failed to update unread message count")
	}

	// res
	log.Printf("message saved succefully from %s to %s", msg.FromUserID, msg.ToUserID)
	return nil

}

// === utility methods == //
func (h *ChatHub) GetOnlineUsers() map[string]kafka.UserStatus {
	return h.statusManager.GetOnlineUsers()
}

func (h *ChatHub) GetConnectedUserIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]string, 0, len(h.clients))
	for userID := range h.clients {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

func (h *ChatHub) PublishUserStatus(userID, nodeID string, online bool) error {
	status := kafka.UserStatus{
		UserID:    userID,
		NodeID:    nodeID,
		Online:    online,
		Timestamp: time.Now().Unix(),
	}

	return h.statusManager.PublishStatus(status)
}

func (h *ChatHub) Close() {
	log.Println("Closing chat hub..")
	if h.messageManager != nil {
		h.messageManager.Close()
	}
	if h.statusManager != nil {
		h.statusManager.Close()
	}
	log.Println("Chat hub closed")
}