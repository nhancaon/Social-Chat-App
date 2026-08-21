package jobs

import (
	"Server/database"
	"Server/models"
	"Server/storage"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// trashRetention controls how long a soft-deleted file stays recoverable before
// this job deletes it for good — same "empty trash after N days" behavior as
// Google Drive.
func trashRetention() time.Duration {
	if raw := os.Getenv("TRASH_RETENTION_HOURS"); raw != "" {
		if hours, err := strconv.Atoi(raw); err == nil {
			return time.Duration(hours) * time.Hour
		}
	}
	return 30 * 24 * time.Hour
}

func TrashPurge(ctx context.Context) error {
	FileSchema := database.DB.Collection("files")
	UserSchema := database.DB.Collection("users")
	cutoff := time.Now().Add(-trashRetention())

	cursor, err := FileSchema.Find(ctx, bson.M{
		"isDeleted": true,
		"deletedAt": bson.M{"$lt": cutoff},
	})
	if err != nil {
		return fmt.Errorf("query files eligible for purge: %w", err)
	}
	defer cursor.Close(ctx)

	var files []models.FileModel
	if err := cursor.All(ctx, &files); err != nil {
		return fmt.Errorf("decode files eligible for purge: %w", err)
	}

	purged, failed := 0, 0
	for _, file := range files {
		if err := storage.DeleteObject(ctx, file.S3Key); err != nil {
			log.Printf("trash-purge: failed to delete %s from S3: %v", file.S3Key, err)
			failed++
			continue
		}

		if _, err := FileSchema.DeleteOne(ctx, bson.M{"_id": file.ID}); err != nil {
			log.Printf("trash-purge: deleted %s from S3 but failed to remove Mongo record: %v", file.ID.Hex(), err)
			failed++
			continue
		}

		// credit the quota back now that the object is actually gone, not when it
		// was first trashed — matches Drive's "trash still counts against quota"
		if ownerID, err := primitive.ObjectIDFromHex(file.OwnerID); err == nil {
			if _, err := UserSchema.UpdateOne(ctx, bson.M{"_id": ownerID}, bson.M{"$inc": bson.M{"storageUsedBytes": -file.SizeBytes}}); err != nil {
				log.Printf("trash-purge: failed to credit back quota for user %s: %v", file.OwnerID, err)
			}
		}
		purged++
	}

	log.Printf("trash-purge: done — %d purged, %d failed, %d eligible", purged, failed, len(files))
	return nil
}
