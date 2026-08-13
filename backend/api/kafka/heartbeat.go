package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type NodeHeartbeat struct {
	NodeID    string   `json:"node_id"`
	TimeStamp int64    `json:"timestamp"`
	UserIDs   []string `json:"user_ids"`
}

type ChatHub interface {
	GetConnectedUserIDs() []string
	PublishUserStatus(userID, nodeID string, online bool) error
}

type HeartbeatManager struct {
	mu                  sync.RWMutex
	wg                  sync.WaitGroup
	nodeID              string
	hub                 ChatHub
	kafkaWriter         *kafka.Writer
	heartbeatReader     *kafka.Reader
	nodeLastSeen        map[string]int64
	processedDeadNodes  map[string]bool
	nodeUsers           map[string][]string
	heartbeatInterval   time.Duration
	healthCheckInterval time.Duration
	timeoutThreshold    time.Duration
	ctx                 context.Context
	cancel              context.CancelFunc
}

func NewHeartbeatManager(kafkaAddr, nodeID string, hub ChatHub) (*HeartbeatManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	km := NewKafkaManager(kafkaAddr)
	if err := km.EnsureTopic([]string{"node-heartbeat"}); err != nil {
		log.Printf("Warning: could not ensure heartbeat topic exists: %v", err)
	}

	time.Sleep(1 * time.Second)

	writer := NewWriter(kafkaAddr, "node-heartbeat")
	reader := NewGroupReader(kafkaAddr, "node-heartbeat", "heartbeat-monitor-"+nodeID)

	hm := &HeartbeatManager{
		nodeID:              nodeID,
		hub:                 hub,
		kafkaWriter:         writer,
		heartbeatReader:     reader,
		nodeLastSeen:        make(map[string]int64),
		nodeUsers:           make(map[string][]string),
		processedDeadNodes:  make(map[string]bool),
		heartbeatInterval:   5 * time.Second,
		healthCheckInterval: 5 * time.Second,
		timeoutThreshold:    15 * time.Second,
		ctx:                 ctx,
		cancel:              cancel,
	}

	log.Printf("[%s] rebuilding node state from heartbeats..", nodeID)
	if err := hm.rebuildNodeStateFromAllPartitions(kafkaAddr); err != nil {
		log.Printf("[%s] node state rebuilding has issues: %v", nodeID, err)
	}

	hm.wg.Add(4)
	go func() { defer hm.wg.Done(); hm.sendHeartbeats() }()
	go func() { defer hm.wg.Done(); hm.listenToHeartBeats() }()
	go func() { defer hm.wg.Done(); hm.monitorNodeHealth() }()
	go func() { defer hm.wg.Done(); hm.cleanupOldDeadNodes() }()

	log.Printf("Heartbeat manager initialized for node %s", nodeID)
	return hm, nil
}

func (hm *HeartbeatManager) rebuildNodeStateFromAllPartitions(kafkaAddr string) error {
	log.Printf("[%s] reading all heartbeat messages from all partitions..", hm.nodeID)

	totalProcessed, err := ReadAllPartitions(kafkaAddr, "node-heartbeat", func(msg kafka.Message) error {
		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return nil
		}

		if event.Type == "node_heartbeat" {
			var heartbeat NodeHeartbeat
			if err := json.Unmarshal(event.Data, &heartbeat); err != nil {
				return nil
			}

			hm.mu.Lock()
			hm.nodeLastSeen[heartbeat.NodeID] = heartbeat.TimeStamp
			hm.nodeUsers[heartbeat.NodeID] = heartbeat.UserIDs
			hm.mu.Unlock()
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[%s] Node state rebuilding completed: %d total heartbeats", hm.nodeID, totalProcessed)
	hm.mu.RLock()
	for nodeID, lastSeen := range hm.nodeLastSeen {
		users := hm.nodeUsers[nodeID]
		log.Printf("[%s] Known node: %s (last seen: %d, users: %d)", hm.nodeID, nodeID, lastSeen, len(users))
	}
	hm.mu.RUnlock()

	return nil
}

func (hm *HeartbeatManager) sendHeartbeats() {
	ticker := time.NewTicker(hm.heartbeatInterval)
	defer ticker.Stop()

	log.Printf("Starting heartbeat sender for node %s", hm.nodeID)

	for {
		select {
		case <-hm.ctx.Done():
			log.Println("Heartbeat sender stopped")
			return
		case <-ticker.C:
			if err := hm.publishHeartbeat(); err != nil {
				log.Printf("Failed to publish heartbeat: %s", err)
			}
		}
	}
}

func (hm *HeartbeatManager) publishHeartbeat() error {
	userIDs := hm.hub.GetConnectedUserIDs()

	heartbeat := NodeHeartbeat{
		NodeID:    hm.nodeID,
		TimeStamp: time.Now().Unix(),
		UserIDs:   userIDs,
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		return err
	}

	event := Event{
		Type: "node_heartbeat",
		Data: json.RawMessage(data),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := kafka.Message{
		Key:   []byte(hm.nodeID),
		Value: eventBytes,
	}

	if err := hm.kafkaWriter.WriteMessages(ctx, msg); err != nil {
		return err
	}

	log.Printf("Heartbeat sent: node=%s, users=%d", hm.nodeID, len(userIDs))

	return nil
}

func (hm *HeartbeatManager) listenToHeartBeats() {
	log.Println("Heartbeat listener started")

	ConsumeLoop(hm.ctx, hm.heartbeatReader, "Heartbeat", func(msg kafka.Message) {
		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshaling heartbeat event: %v", err)
			return
		}

		if event.Type == "node_heartbeat" {
			var heartbeat NodeHeartbeat
			if err := json.Unmarshal(event.Data, &heartbeat); err != nil {
				log.Printf("Error unmarshaling heartbeat: %v", err)
				return
			}

			hm.handleHeartBeat(heartbeat)
		}
	})
}

func (hm *HeartbeatManager) handleHeartBeat(heartbeat NodeHeartbeat) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.nodeLastSeen[heartbeat.NodeID] = heartbeat.TimeStamp
	hm.nodeUsers[heartbeat.NodeID] = heartbeat.UserIDs

	if hm.processedDeadNodes[heartbeat.NodeID] {
		log.Printf("Node %s is alive again! removing from dead list", heartbeat.NodeID)
		delete(hm.processedDeadNodes, heartbeat.NodeID)
	}

	log.Printf("Heartbeat received: node=%s, users=%d, timestamp=%d", heartbeat.NodeID, len(heartbeat.UserIDs), heartbeat.TimeStamp)
}

func (hm *HeartbeatManager) monitorNodeHealth() {
	ticker := time.NewTicker(hm.healthCheckInterval)
	defer ticker.Stop()

	log.Println("Node health monitor started")

	for {
		select {
		case <-hm.ctx.Done():
			log.Println("Node health monitor stopped")
			return
		case <-ticker.C:
			hm.checkDeadNodes()
		}
	}
}

func (hm *HeartbeatManager) checkDeadNodes() {
	now := time.Now().Unix()
	timeoutSeconds := int64(hm.timeoutThreshold.Seconds())

	hm.mu.Lock()
	deadNodes := []string{}

	for nodeID, lastSeen := range hm.nodeLastSeen {
		if nodeID == hm.nodeID {
			continue
		}

		timeSinceLastSeen := now - lastSeen
		if timeSinceLastSeen > timeoutSeconds {
			if !hm.processedDeadNodes[nodeID] {
				deadNodes = append(deadNodes, nodeID)
				log.Printf("Node %s is dead (last seen %d seconds ago)", nodeID, timeSinceLastSeen)
			}
		}
	}

	hm.mu.Unlock()
	if len(deadNodes) > 0 {
		if hm.shouldHandleFailure() {
			for _, nodeID := range deadNodes {
				hm.HandleDeadNode(nodeID)
			}
		} else {
			log.Printf("Not handling dead nodes %v - another node is responsible", deadNodes)
		}
	}
}

func (hm *HeartbeatManager) shouldHandleFailure() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	aliveNodes := []string{hm.nodeID}
	now := time.Now().Unix()
	timeoutSeconds := int64(hm.timeoutThreshold.Seconds())

	for nodeID, lastSeen := range hm.nodeLastSeen {
		if nodeID == hm.nodeID {
			continue
		}
		timeSinceLastSeen := now - lastSeen
		if timeSinceLastSeen <= timeoutSeconds {
			aliveNodes = append(aliveNodes, nodeID)
		}
	}

	sort.Strings(aliveNodes)

	designatedNode := aliveNodes[0]
	isDesignated := designatedNode == hm.nodeID

	if isDesignated {
		log.Printf("This node (%s) is designated to handle dead nodes (alive nodes: %v)", hm.nodeID, aliveNodes)
	}
	return isDesignated
}

func (hm *HeartbeatManager) HandleDeadNode(nodeID string) {
	hm.mu.Lock()
	userIDs := hm.nodeUsers[nodeID]
	hm.processedDeadNodes[nodeID] = true
	hm.mu.Unlock()

	if len(userIDs) == 0 {
		log.Printf("Dead node %s had no users", nodeID)
		return
	}

	log.Printf("Handling dead node %s: marking %d users as offline.", nodeID, len(userIDs))

	for _, userID := range userIDs {
		if err := hm.hub.PublishUserStatus(userID, nodeID, false); err != nil {
			log.Printf("Failed to publish offline status for user %s: %v", userID, err)
		} else {
			log.Printf("User %s marked offline (from dead node: %s)", userID, nodeID)
		}
	}
}

func (hm *HeartbeatManager) cleanupOldDeadNodes() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Dead node cleanup service started")

	for {
		select {
		case <-hm.ctx.Done():
			log.Println("Dead node cleanup service stopped")
			return
		case <-ticker.C:
			now := time.Now().Unix()
			cleanupThreshold := int64(5 * 60)

			hm.mu.Lock()

			nodesToCleanup := []string{}
			for nodeID, lastSeen := range hm.nodeLastSeen {
				if nodeID == hm.nodeID {
					continue
				}

				timeSinceLastSeen := now - lastSeen
				if timeSinceLastSeen > cleanupThreshold && hm.processedDeadNodes[nodeID] {
					nodesToCleanup = append(nodesToCleanup, nodeID)
				}
			}

			for _, nodeID := range nodesToCleanup {
				log.Printf("forgetting old dead node: %s (last seen %d seconds ago)", nodeID, now-hm.nodeLastSeen[nodeID])
				delete(hm.nodeLastSeen, nodeID)
				delete(hm.nodeUsers, nodeID)
				delete(hm.processedDeadNodes, nodeID)
			}

			hm.mu.Unlock()

			if len(nodesToCleanup) > 0 {
				log.Printf("Cleaned up %d old dead nodes", len(nodesToCleanup))
			}
		}
	}
}

// Stop cancels all background goroutines and waits for them to exit before
// closing the kafka writer/reader, so nothing reads from a closed reader.
func (hm *HeartbeatManager) Stop() {
	log.Println("Stopping heartbeat manager..")
	hm.cancel()
	hm.wg.Wait()

	if hm.kafkaWriter != nil {
		hm.kafkaWriter.Close()
	}
	if hm.heartbeatReader != nil {
		hm.heartbeatReader.Close()
	}
	log.Println("Heartbeat manager stopped")
}
