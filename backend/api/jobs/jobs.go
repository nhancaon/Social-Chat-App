package jobs

import (
	"context"
	"fmt"
)

// Run dispatches a one-shot background job by name. Each job connects to Mongo/S3
// (already initialized by main.go before calling Run) and exits — these are meant
// to run as K8s CronJob pods using the same binary/image as the HTTP server, just
// with a different command.
func Run(ctx context.Context, name string) error {
	switch name {
	case "archive-scan":
		return ArchiveScan(ctx)
	case "trash-purge":
		return TrashPurge(ctx)
	case "restore-poll":
		return RestorePoll(ctx)
	case "abandoned-upload-purge":
		return AbandonedUploadPurge(ctx)
	default:
		return fmt.Errorf("unknown job %q (want archive-scan | trash-purge | restore-poll | abandoned-upload-purge)", name)
	}
}
