package jobs

import (
	"Server/database"
	"Server/kafka"
	"Server/models"
	"Server/storage"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RestorePoll checks every GLACIER file with a pending restore request. S3 has no
// way to push "restore complete" to the app, so this job polls HeadObject instead —
// the tradeoff called out in the design: archival can lean on S3's own Lifecycle
// rule, but restore completion genuinely needs orchestration the app owns.
func RestorePoll(ctx context.Context) error {
	FileSchema := database.DB.Collection("files")
	NotificationSchema := database.DB.Collection("notifications")

	cursor, err := FileSchema.Find(ctx, bson.M{"storageClass": models.StorageClassRestoring})
	if err != nil {
		return fmt.Errorf("query files pending restore: %w", err)
	}
	defer cursor.Close(ctx)

	var files []models.FileModel
	if err := cursor.All(ctx, &files); err != nil {
		return fmt.Errorf("decode files pending restore: %w", err)
	}
	if len(files) == 0 {
		log.Println("restore-poll: nothing pending")
		return nil
	}

	kafkaAddr := os.Getenv("KAFKA_ADDR")
	if kafkaAddr == "" {
		kafkaAddr = "127.0.0.1:29092"
	}
	writer := kafka.NewWriter(kafkaAddr, "notifications")
	defer writer.Close()

	completed := 0
	for _, file := range files {
		done, err := storage.IsRestoreComplete(ctx, file.S3Key)
		if err != nil {
			log.Printf("restore-poll: HeadObject failed for %s: %v", file.S3Key, err)
			continue
		}
		if !done {
			continue
		}

		if _, err := FileSchema.UpdateOne(ctx, bson.M{"_id": file.ID}, bson.M{"$set": bson.M{
			"storageClass":     models.StorageClassRestored,
			"restoreExpiresAt": time.Now().Add(7 * 24 * time.Hour),
		}}); err != nil {
			log.Printf("restore-poll: failed to update Mongo for %s: %v", file.ID.Hex(), err)
			continue
		}

		notifyUser(ctx, writer, NotificationSchema, file)
		completed++
	}

	log.Printf("restore-poll: done — %d completed, %d checked", completed, len(files))
	return nil
}

// notifyUser saves a normal notification record (so it's still there later for an
// offline user) and publishes onto the same Kafka "notifications" topic every
// backend replica already consumes via its own per-node consumer group (see
// realtime/notify_hub.go) — the live push reaches the user over the existing
// WebSocket delivery path without this one-shot job running a WebSocket server
// of its own.
func notifyUser(ctx context.Context, writer *kafkago.Writer, NotificationSchema *mongo.Collection, file models.FileModel) {
	notification := models.Notification{
		Details:   fmt.Sprintf("Your file \"%s\" has been restored and is ready to download.", file.Filename),
		MainUID:   file.OwnerID,
		TargetID:  file.ID.Hex(),
		UserID:    file.OwnerID,
		User:      models.User{Name: "System"},
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	res, err := NotificationSchema.InsertOne(ctx, notification)
	if err != nil {
		log.Printf("restore-poll: failed to save notification for file %s: %v", file.ID.Hex(), err)
		return
	}
	notification.ID = res.InsertedID.(primitive.ObjectID)

	payload := kafka.Notification{
		ID:        notification.ID.Hex(),
		Details:   notification.Details,
		MainUID:   notification.MainUID,
		TargetID:  notification.TargetID,
		UserID:    notification.UserID,
		IsRead:    notification.IsRead,
		CreatedAt: notification.CreatedAt,
		User:      kafka.User{Name: notification.User.Name, Avatar: notification.User.Avatar},
	}

	dataBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("restore-poll: failed to marshal notification for file %s: %v", file.ID.Hex(), err)
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := writer.WriteMessages(writeCtx, kafkago.Message{
		Key:   []byte(file.OwnerID),
		Value: dataBytes,
	}); err != nil {
		log.Printf("restore-poll: failed to publish live notification for file %s: %v", file.ID.Hex(), err)
	}
}
