package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Message struct {
	FromUserID string `json:"from_user_id"`
	ToUserID   string `json:"to_user_id"`
	Content    string `json:"content"`
	Timestamp  int64  `json:"timestamp"`
	MessageID  string `json:"message_id,omitempty"`
}

type MessageCache struct {
	messages map[string]int64
	mu       sync.RWMutex
}

func NewMessageCache() *MessageCache {
	cache := &MessageCache{
		messages: make(map[string]int64),
	}
	go cache.cleanup()
	return cache
}

func (mc *MessageCache) Add(messageID string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exist := mc.messages[messageID]; exist {
		return false
	}
	mc.messages[messageID] = time.Now().Unix()
	return true
}

func (mc *MessageCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		mc.mu.Lock()
		now := time.Now().Unix()
		for id, ts := range mc.messages {
			if now-ts > 300 {
				delete(mc.messages, id)
			}
		}
		mc.mu.Unlock()
	}
}

type MessageHandler interface {
	DeliverMessage(msg *Message)
}

type MessageManager struct {
	kafkaWriter   *kafka.Writer
	messageReader *kafka.Reader
	messageCache  *MessageCache
	handler       MessageHandler
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewMessageManager(kafkaAddr, nodeID string, handler MessageHandler) (*MessageManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	km := NewKafkaManager(kafkaAddr)
	if err := km.EnsureTopic([]string{"chat-messages"}); err != nil {
		log.Printf("Warning: could not ensure chat-messages topic exists: %v", err)
	}

	time.Sleep(1 * time.Second)

	writer := NewWriter(kafkaAddr, "chat-messages")
	reader := NewGroupReader(kafkaAddr, "chat-messages", "chat-group-"+nodeID)

	mm := &MessageManager{
		kafkaWriter:   writer,
		messageReader: reader,
		messageCache:  NewMessageCache(),
		handler:       handler,
		ctx:           ctx,
		cancel:        cancel,
	}

	go mm.listenToMessages()
	log.Println("Message Manager Initialized")
	return mm, nil
}

func (mm *MessageManager) PublishMessage(msg *Message) error {
	msg.MessageID = fmt.Sprintf("%s-%s-%d", msg.FromUserID, msg.ToUserID, msg.Timestamp)

	dataBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	event := Event{
		Type: "message",
		Data: json.RawMessage(dataBytes),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	convKey := GetConvKey(msg.FromUserID, msg.ToUserID)
	kafkaMsg := kafka.Message{
		Key:   []byte(convKey),
		Value: eventBytes,
	}

	return mm.kafkaWriter.WriteMessages(ctx, kafkaMsg)
}

func (mm *MessageManager) SendMessageWithRetry(msg *Message, maxRetries int) error {
	msg.Timestamp = time.Now().Unix()

	for i := 0; i < maxRetries; i++ {
		err := mm.PublishMessage(msg)
		if err == nil {
			return nil
		}
		log.Printf("Retry %d faild for message: %v", i+1, err)
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}
	return fmt.Errorf("faild to send message after %d retries", maxRetries)
}

func (mm *MessageManager) listenToMessages() {
	log.Println("chat messages kafka listener started")

	ConsumeLoop(mm.ctx, mm.messageReader, "chat messages", func(msg kafka.Message) {
		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			return
		}

		if event.Type == "message" {
			var chatMsg Message
			if err := json.Unmarshal(event.Data, &chatMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				return
			}

			if !mm.messageCache.Add(chatMsg.MessageID) {
				log.Printf("Duplicate message ignored: %s", chatMsg.MessageID)
				return
			}

			log.Printf("Processing new message from %s to %s", chatMsg.FromUserID, chatMsg.ToUserID)
			mm.handler.DeliverMessage(&chatMsg)
		}
	})
}

// close colces the message manager
func (mm *MessageManager) Close() {
	log.Println("Stopping message manger..")
	mm.cancel()

	if mm.kafkaWriter != nil {
		mm.kafkaWriter.Close()
	}

	if mm.messageReader != nil {
		mm.messageReader.Close()
	}
	log.Println("message manger stopped")
}

// helper
func GetConvKey(userID1, userID2 string) string {
	if userID1 > userID2 {
		return userID2 + "-" + userID1
	}
	return userID1 + "-" + userID2
}