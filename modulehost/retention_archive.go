package modulehost

import (
	"context"
	"time"
)

const RetentionArchiveOwnerNotification = "notification"

type RetentionArchiveJob struct {
	ID          string
	WorkspaceID string
	ArchivedAt  time.Time
}

type RetentionArchivePolicy struct {
	Key     string
	Version string
}

// RetentionArchiveStore is the narrow cross-owner write capability for the
// shared Lifecycle archive. Notification never receives Lifecycle's backing
// table or persistence implementation.
type RetentionArchiveStore interface {
	Archived(context.Context, string, string, string, string) (bool, error)
	ArchivePayload(context.Context, string, RetentionArchiveJob, RetentionArchivePolicy, string, string, []byte) (bool, error)
}
