package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type UserStatus struct {
	UserID    string `json:"user_id"`
	NodeID    string `json:"node_id"`
	Online    bool   `json:"online"`
	Timestamp int64  `json:"timestamp"`
}

type StatusHandler interface {
	HandlerUserStatus(status UserStatus)
}

type StatusManager struct {
	mu           sync.RWMutex
	nodeID       string
	kafkaWriter  *kafka.Writer
	statusReader *kafka.Reader
	onlineUsers  map[string]UserStatus
	handler      StatusHandler
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewStatusManager(kafkaAddr, nodeID string, handler StatusHandler) (*StatusManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	km := NewKafkaManager(kafkaAddr)
	if err := km.EnsureTopic([]string{"user-status"}); err != nil {
		log.Printf("Warning: could not ensure user-status topic exists: %v", err)
	}

	time.Sleep(1 * time.Second)

	writer := NewWriter(kafkaAddr, "user-status")
	reader := NewGroupReader(kafkaAddr, "user-status", "status-group-"+nodeID)

	sm := &StatusManager{
		nodeID:       nodeID,
		kafkaWriter:  writer,
		statusReader: reader,
		onlineUsers:  make(map[string]UserStatus),
		handler:      handler,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Rebuild state form all partitions
	log.Printf("%s Rebuling user status form kafak cluser..", nodeID)
	if err := sm.rebuildStatusFromAllPartitions(kafkaAddr); err != nil {
		log.Printf("%s status rebuilding had isses: %v", nodeID, err)
	}
	go sm.listenToStatus()

	log.Printf("%s Status manager ready - %d users online", nodeID, sm.countOnlineUsers())

	return sm, nil
}

func (sm *StatusManager) PublishStatus(status UserStatus) error {
	dataBytes, err := json.Marshal(status)
	if err != nil {
		return err
	}

	event := Event{
		Type: "user_status",
		Data: json.RawMessage(dataBytes),
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := kafka.Message{
		Key:   []byte(status.UserID),
		Value: eventBytes,
	}

	return sm.kafkaWriter.WriteMessages(ctx, msg)
}

func (sm *StatusManager) listenToStatus() {
	log.Println("User status kafka listener started")

	ConsumeLoop(sm.ctx, sm.statusReader, "user status", func(msg kafka.Message) {
		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			return
		}

		if event.Type == "user_status" {
			var status UserStatus
			if err := json.Unmarshal(event.Data, &status); err != nil {
				log.Printf("Error unmarshaling user status: %v", err)
				return
			}

			sm.mu.Lock()
			sm.onlineUsers[status.UserID] = status
			sm.mu.Unlock()

			log.Printf("User %s status: online=%v (node: %s)", status.UserID, status.Online, status.NodeID)

			if sm.handler != nil {
				sm.handler.HandlerUserStatus(status)
			} else {
				log.Printf("Warning: handler is nil, cannot broadcast status for user %s", status.UserID)
			}
		}
	})
}

func (sm *StatusManager) rebuildStatusFromAllPartitions(kafakAddr string) error {
	log.Printf("%s Reading All User Status messages form all Partitions..", sm.nodeID)

	totalProcessed := 0
	onlineCount := 0

	processed, err := ReadAllPartitions(kafakAddr, "user-status", func(msg kafka.Message) error {
		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return nil
		}

		if event.Type == "user_status" {
			var status UserStatus
			if err := json.Unmarshal(event.Data, &status); err != nil {
				return nil
			}

			sm.mu.Lock()
			sm.onlineUsers[status.UserID] = status
			sm.mu.Unlock()

			if status.Online {
				onlineCount++
			}
		}
		return nil
	})

	totalProcessed = processed
	if err != nil {
		return err
	}

	log.Printf("%s state rebuilding complted :%d total messages, %d online users", sm.nodeID, totalProcessed, onlineCount)

	sm.mu.RLock()
	for userID, status := range sm.onlineUsers {
		if status.Online {
			log.Printf("%s Online User : %s no node %s", sm.nodeID, userID, status.NodeID)
		}
	}
	sm.mu.RUnlock()

	return nil
}

func (sm *StatusManager) countOnlineUsers() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	count := 0
	for _, status := range sm.onlineUsers {
		if status.Online {
			count++
		}
	}
	return count
}

func (sm *StatusManager) GetOnlineUsers() map[string]UserStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	users := make(map[string]UserStatus)
	for userId, status := range sm.onlineUsers {
		if status.Online {
			users[userId] = status
		}
	}
	return users
}

func (sm *StatusManager) Close() {
	log.Println("Stopping stauts manager...")
	sm.cancel()

	if sm.kafkaWriter != nil {
		sm.kafkaWriter.Close()
	}
	if sm.statusReader != nil {
		sm.statusReader.Close()
	}

	log.Println("Status manager stopped")
}
