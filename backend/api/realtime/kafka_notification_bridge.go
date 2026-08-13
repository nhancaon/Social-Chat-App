package realtime

import (
	"Server/kafka"
	"context"
	"encoding/json"
	"log"
	"time"

	skafka "github.com/segmentio/kafka-go"
)

type KafkaNotificationBridge struct {
	writer  *skafka.Writer
	reader  *skafka.Reader
	ctx     context.Context
	cancel  context.CancelFunc
	handler NotificationDeliveryHandler
}

type NotificationDeliveryHandler interface {
	DeliverToLocalClient(userID string, notification Notification)
}

func NewKafkaNotificationBridge(kafkaAddr, nodeID string) (*KafkaNotificationBridge, error) {
	ctx, cancel := context.WithCancel(context.Background())

	writer := kafka.NewWriter(kafkaAddr, "notifications")
	reader := kafka.NewGroupReader(kafkaAddr, "notifications", "notification-group-"+nodeID)

	bridge := &KafkaNotificationBridge{
		writer: writer,
		reader: reader,
		ctx:    ctx,
		cancel: cancel,
	}

	go bridge.consumeNotification()
	log.Println("kafka notification bridge initialized")
	return bridge, nil
}

func (k *KafkaNotificationBridge) SetDeliveryHandler(handler NotificationDeliveryHandler) {
	k.handler = handler
}

func (k *KafkaNotificationBridge) PublishNotification(targetUID string, notification Notification) error {
	dataBytes, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := skafka.Message{
		Key:   []byte(targetUID),
		Value: dataBytes,
	}

	if err := k.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("failed to publish notification to kafka: %v", err)
	}

	return nil
}

func (k *KafkaNotificationBridge) consumeNotification() {
	log.Println("kafka notification consumer started")

	kafka.ConsumeLoop(k.ctx, k.reader, "notification", func(msg skafka.Message) {
		var notification Notification
		if err := json.Unmarshal(msg.Value, &notification); err != nil {
			log.Printf("Error unmarshaling notification: %v", err)
			return
		}

		if k.handler != nil {
			k.handler.DeliverToLocalClient(notification.MainUID, notification)
		}
	})
}

func (k *KafkaNotificationBridge) Close() {
	log.Println("Closing kafka notification bridge...")
	k.cancel()

	if k.writer != nil {
		k.writer.Close()
	}
	if k.reader != nil {
		k.reader.Close()
	}
	log.Println("kafka notification bridge closed")
}
