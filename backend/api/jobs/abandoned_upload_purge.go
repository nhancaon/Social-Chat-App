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

// abandonedUploadAfter controls how long a "pending" upload record (client got
// a presigned URL but never called /confirm) is kept before being purged.
// Default is much shorter than trash retention — these records have no
// user-facing value at all, they're pure bookkeeping cruft from an upload
// that never finished (e.g. CORS/network failure between the PUT and confirm).
func abandonedUploadAfter() time.Duration {
	if raw := os.Getenv("ABANDONED_UPLOAD_AFTER_HOURS"); raw != "" {
		if hours, err := strconv.Atoi(raw); err == nil {
			return time.Duration(hours) * time.Hour
		}
	}
	return 1 * time.Hour
}

func AbandonedUploadPurge(ctx context.Context) error {
	FileSchema := database.DB.Collection("files")
	cutoff := time.Now().Add(-abandonedUploadAfter())

	cursor, err := FileSchema.Find(ctx, bson.M{
		"uploaded":  false,
		"createdAt": bson.M{"$lt": cutoff},
	})
	if err != nil {
		return fmt.Errorf("query abandoned uploads: %w", err)
	}
	defer cursor.Close(ctx)

	var files []models.FileModel
	if err := cursor.All(ctx, &files); err != nil {
		return fmt.Errorf("decode abandoned uploads: %w", err)
	}

	purged := 0
	for _, file := range files {
		// Best-effort: normally no S3 object exists behind an unconfirmed
		// upload, but a client can legitimately finish the S3 PUT and then
		// fail to call /confirm (e.g. dropped connection right after). Calling
		// DeleteObject on a key that was never written is a no-op in S3 (not
		// an error), so this costs nothing in the common case and closes that
		// leak in the rare one.
		if err := storage.DeleteObject(ctx, file.S3Key); err != nil {
			log.Printf("abandoned-upload-purge: failed to delete possible orphan object %s: %v", file.S3Key, err)
			continue
		}
		if _, err := FileSchema.DeleteOne(ctx, bson.M{"_id": file.ID}); err != nil {
			log.Printf("abandoned-upload-purge: failed to remove record %s: %v", file.ID.Hex(), err)
			continue
		}
		purged++
	}

	log.Printf("abandoned-upload-purge: done — %d purged, %d eligible", purged, len(files))
	return nil
}
