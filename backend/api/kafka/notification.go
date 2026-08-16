package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Notification struct {
	ID        string    `json:"_id"`
	Details   string    `json:"details"`
	MainUID   string    `json:"mainuid"`
	TargetID  string    `json:"targetid"`
	IsReaded  bool      `json:"isreded"`
	CreatedAt time.Time `json:"createdAt"`
	User      User      `json:"user"`
}

type User struct {
	Name     string `json:"name"`
	ImageUrl string `json:"imageUrl"`
}

type NotificationHandler interface {
	DeliverToLocalClient(userID string, notification Notification)
}

type NotificationManager struct {
	writer  *kafka.Writer
	reader  *kafka.Reader
	ctx     context.Context
	cancel  context.CancelFunc
	handler NotificationHandler
}

func NewNotificationManager(kafkaAddr, nodeID string, handler NotificationHandler) (*NotificationManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	km := NewKafkaManager(kafkaAddr)
	if err := km.EnsureTopic([]string{"notifications"}); err != nil {
		log.Printf("Warning: could not ensure notifications topic exists: %v", err)
	}

	time.Sleep(1 * time.Second)

	writer := NewWriter(kafkaAddr, "notifications")
	reader := NewGroupReader(kafkaAddr, "notifications", "notification-group-"+nodeID)

	nm := &NotificationManager{
		writer:  writer,
		reader:  reader,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}

	go nm.consumeNotification()
	log.Println("Notification Manager Initialized")
	return nm, nil
}

func (nm *NotificationManager) PublishNotification(targetUID string, notification Notification) error {
	dataBytes, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := kafka.Message{
		Key:   []byte(targetUID),
		Value: dataBytes,
	}

	if err := nm.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("failed to publish notification to kafka: %v", err)
	}

	return nil
}

func (nm *NotificationManager) consumeNotification() {
	log.Println("notifications kafka listener started")

	ConsumeLoop(nm.ctx, nm.reader, "notification", func(msg kafka.Message) {
		var notification Notification
		if err := json.Unmarshal(msg.Value, &notification); err != nil {
			log.Printf("Error unmarshaling notification: %v", err)
			return
		}

		if nm.handler != nil {
			nm.handler.DeliverToLocalClient(notification.MainUID, notification)
		}
	})
}

func (nm *NotificationManager) Close() {
	log.Println("Stopping notification manager...")
	nm.cancel()

	if nm.writer != nil {
		nm.writer.Close()
	}
	if nm.reader != nil {
		nm.reader.Close()
	}
	log.Println("notification manager stopped")
}
