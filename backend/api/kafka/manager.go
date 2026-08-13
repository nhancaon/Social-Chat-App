package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type KafkaManager struct {
	addr string
}

func NewKafkaManager(addr string) *KafkaManager {
	return &KafkaManager{addr: addr}
}

func (km *KafkaManager) EnsureTopic(topics []string) error {
	conn, err := kafka.Dial("tcp", km.addr)
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %v", err)
	}

	connController, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %v", err)
	}
	defer connController.Close()

	topicConfigs := []kafka.TopicConfig{}
	for _, topic := range topics {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		})
	}

	return connController.CreateTopics(topicConfigs...)
}

// NewWriter returns a kafka.Writer with this project's standard publish
// settings, fixed to a single topic.
func NewWriter(addr, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(addr),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		BatchTimeout:           10 * time.Millisecond,
		WriteTimeout:           10 * time.Second,
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
	}
}

// NewGroupReader returns a kafka.Reader in its own consumer group, so it
// receives every message on the topic instead of sharing partitions with
// other readers in a different group.
func NewGroupReader(addr, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{addr},
		Topic:          topic,
		GroupID:        groupID,
		StartOffset:    kafka.LastOffset,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
}

// ConsumeLoop reads messages from reader until ctx is cancelled, calling
// handle for each one. Read errors are logged under label and retried after 1s.
func ConsumeLoop(ctx context.Context, reader *kafka.Reader, label string, handle func(kafka.Message)) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("%s listener stopped", label)
			return
		default:
		}

		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Printf("%s read error: %v, retrying in 1s", label, err)
			time.Sleep(1 * time.Second)
			continue
		}

		handle(msg)
	}
}

func (km *KafkaManager) HealthCheck() error {
	conn, err := kafka.Dial("tcp", km.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Brokers()
	return err
}

func WaitForKafka(addr string, timeout time.Duration) error {
	km := NewKafkaManager(addr)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := km.HealthCheck(); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("kafka not ready after %v", timeout)
}

// ReadAllPartitions reads every partition of topic concurrently, from offset 0
// up to a 15s per-partition idle timeout, invoking handler for each message.
// handler may be called concurrently from multiple partitions, so it must be
// safe for concurrent use.
func ReadAllPartitions(addr string, topic string, handler func(msg kafka.Message) error) (int, error) {
	log.Printf("Reading all partitions for topic: %s", topic)

	conn, err := kafka.Dial("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to kafka: %v", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return 0, fmt.Errorf("failed to read partitions: %v", err)
	}

	log.Printf("Found %d partitions for the topic %s", len(partitions), topic)

	var (
		mu             sync.Mutex
		wg             sync.WaitGroup
		totalProcessed int
		firstErr       error
	)

	for _, partition := range partitions {
		wg.Add(1)
		go func(partitionID int) {
			defer wg.Done()

			processed, err := readPartition(addr, topic, partitionID, handler)

			mu.Lock()
			defer mu.Unlock()
			totalProcessed += processed
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}(partition.ID)
	}

	wg.Wait()

	log.Printf("Completed reading all partitions for %s: %d total messages", topic, totalProcessed)
	return totalProcessed, firstErr
}

func readPartition(addr, topic string, partitionID int, handler func(msg kafka.Message) error) (int, error) {
	log.Printf("Reading partition %d for topic %s..", partitionID, topic)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{addr},
		Topic:     topic,
		Partition: partitionID,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
		MaxWait:   100 * time.Millisecond,
	})
	defer reader.Close()

	reader.SetOffset(0)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	processed := 0
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				log.Printf("partition %d: timeout reached after processing %d messages", partitionID, processed)
			} else {
				log.Printf("partition %d: read error after %d messages: %v", partitionID, processed, err)
			}
			break
		}

		if err := handler(msg); err != nil {
			return processed, err
		}

		processed++
	}

	log.Printf("Partition %d: processed %d messages", partitionID, processed)
	return processed, nil
}
