package storage

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	S3Client      *s3.Client
	PresignClient *s3.PresignClient
	BucketName    string
)

func InitS3() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Println("Failed to load AWS config:", err.Error())
		return
	}

	S3Client = s3.NewFromConfig(cfg)
	PresignClient = s3.NewPresignClient(S3Client)
	BucketName = os.Getenv("AWS_S3_BUCKET")
	log.Println("S3 storage initialized, bucket:", BucketName)
}

func PresignUploadURL(ctx context.Context, key, contentType string, expires time.Duration) (string, error) {
	req, err := PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &BucketName,
		Key:         &key,
		ContentType: &contentType,
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func PresignDownloadURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, err := PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &BucketName,
		Key:    &key,
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// CopyToGlacier re-copies an object onto itself with StorageClass=GLACIER — the
// standard way to change an S3 object's storage class in place. Used by the
// archive-scan job instead of waiting on the bucket's own Lifecycle rule, so the
// archival flow is demoable on a schedule the app controls.
func CopyToGlacier(ctx context.Context, key string) error {
	copySource := fmt.Sprintf("%s/%s", BucketName, url.PathEscape(key))
	_, err := S3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:            &BucketName,
		Key:               &key,
		CopySource:        &copySource,
		StorageClass:      s3types.StorageClassGlacier,
		MetadataDirective: s3types.MetadataDirectiveCopy,
	})
	return err
}

func DeleteObject(ctx context.Context, key string) error {
	_, err := S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &BucketName,
		Key:    &key,
	})
	return err
}

// RequestRestore kicks off an async Glacier rehydration; forDays controls how
// long the restored copy stays available before S3 lets it lapse back to GLACIER.
func RequestRestore(ctx context.Context, key string, forDays int32) error {
	_, err := S3Client.RestoreObject(ctx, &s3.RestoreObjectInput{
		Bucket: &BucketName,
		Key:    &key,
		RestoreRequest: &s3types.RestoreRequest{
			Days: awssdk.Int32(forDays),
			GlacierJobParameters: &s3types.GlacierJobParameters{
				Tier: s3types.TierStandard,
			},
		},
	})
	return err
}

// IsRestoreComplete reports whether a previously requested Glacier restore has
// finished. S3 doesn't push a completion event to the app, so this is a poll:
// HeadObject's Restore header goes from ongoing-request="true" to "false" once
// the rehydrated copy is actually readable.
func IsRestoreComplete(ctx context.Context, key string) (bool, error) {
	out, err := S3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &BucketName,
		Key:    &key,
	})
	if err != nil {
		return false, err
	}
	if out.Restore == nil {
		return false, nil
	}
	return strings.Contains(*out.Restore, `ongoing-request="false"`), nil
}
