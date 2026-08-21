package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	StorageClassStandard  = "STANDARD"
	StorageClassGlacier   = "GLACIER"
	StorageClassRestoring = "RESTORING"
	StorageClassRestored  = "RESTORED"
)

// DefaultStorageQuotaBytes is used whenever a user has no explicit quota set (StorageQuotaBytes == 0),
// so existing users don't need a migration before this feature works.
const DefaultStorageQuotaBytes int64 = 5 * 1024 * 1024 * 1024 // 5GB

type FileModel struct {
	ID             primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	OwnerID        string             `json:"ownerId" bson:"ownerId"`
	Filename       string             `json:"filename" bson:"filename"`
	S3Key          string             `json:"s3Key" bson:"s3Key"`
	SizeBytes      int64              `json:"sizeBytes" bson:"sizeBytes"`
	ContentType    string             `json:"contentType" bson:"contentType"`
	StorageClass   string             `json:"storageClass" bson:"storageClass"`
	Uploaded       bool               `json:"uploaded" bson:"uploaded"`
	UploadedAt     time.Time          `json:"uploadedAt,omitempty" bson:"uploadedAt,omitempty"`
	LastAccessedAt time.Time          `json:"lastAccessedAt,omitempty" bson:"lastAccessedAt,omitempty"`
	ArchivedAt     time.Time          `json:"archivedAt,omitempty" bson:"archivedAt,omitempty"`
	// RestoreRequestedAt/RestoreExpiresAt back the Glacier restore flow: a restored
	// copy is only temporary, after RestoreExpiresAt S3 lets it lapse and the file
	// falls back to GLACIER (the original archived copy is never touched).
	RestoreRequestedAt time.Time `json:"restoreRequestedAt,omitempty" bson:"restoreRequestedAt,omitempty"`
	RestoreExpiresAt   time.Time `json:"restoreExpiresAt,omitempty" bson:"restoreExpiresAt,omitempty"`
	IsDeleted          bool      `json:"isDeleted" bson:"isDeleted"`
	DeletedAt          time.Time `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
	CreatedAt          time.Time `json:"createdAt" bson:"createdAt"`
}

// interfaces
type CreateUploadRequest struct {
	Filename    string `json:"filename" bson:"filename" validate:"required"`
	SizeBytes   int64  `json:"sizeBytes" bson:"sizeBytes" validate:"required,min=1"`
	ContentType string `json:"contentType" bson:"contentType"`
}
