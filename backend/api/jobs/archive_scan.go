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
)

// archiveAfter controls how long a file must sit unaccessed before this job moves
// it to Glacier. The production default (90 days) mirrors the S3 bucket's own
// Lifecycle rule (see terraform/s3.tf) — the CronJob overrides it via env for a
// fast, controllable live demo instead of waiting on AWS's ~daily lifecycle sweep.
func archiveAfter() time.Duration {
	if raw := os.Getenv("ARCHIVE_AFTER_HOURS"); raw != "" {
		if hours, err := strconv.Atoi(raw); err == nil {
			return time.Duration(hours) * time.Hour
		}
	}
	return 90 * 24 * time.Hour
}

func ArchiveScan(ctx context.Context) error {
	FileSchema := database.DB.Collection("files")
	cutoff := time.Now().Add(-archiveAfter())

	cursor, err := FileSchema.Find(ctx, bson.M{
		"uploaded":       true,
		"isDeleted":      false,
		"storageClass":   models.StorageClassStandard,
		"lastAccessedAt": bson.M{"$lt": cutoff},
	})
	if err != nil {
		return fmt.Errorf("query files eligible for archival: %w", err)
	}
	defer cursor.Close(ctx)

	var files []models.FileModel
	if err := cursor.All(ctx, &files); err != nil {
		return fmt.Errorf("decode files eligible for archival: %w", err)
	}

	archived, failed := 0, 0
	for _, file := range files {
		if err := storage.CopyToGlacier(ctx, file.S3Key); err != nil {
			log.Printf("archive-scan: failed to archive %s (%s): %v", file.ID.Hex(), file.S3Key, err)
			failed++
			continue
		}

		if _, err := FileSchema.UpdateOne(ctx, bson.M{"_id": file.ID}, bson.M{"$set": bson.M{
			"storageClass": models.StorageClassGlacier,
			"archivedAt":   time.Now(),
		}}); err != nil {
			log.Printf("archive-scan: archived %s in S3 but failed to update Mongo: %v", file.ID.Hex(), err)
			failed++
			continue
		}
		archived++
	}

	log.Printf("archive-scan: done — %d archived, %d failed, %d eligible", archived, failed, len(files))
	return nil
}
