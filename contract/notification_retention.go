package contract

import (
	"fmt"
	"strings"
	"time"
)

const (
	NotificationRetentionHistoryPolicy     = "notification.history.v1"
	NotificationRetentionPublicationPolicy = "notification.publication_history.v1"
)

type NotificationRetentionPolicy struct {
	Key                     string           `json:"key"`
	Version                 string           `json:"version"`
	DefaultRetentionSeconds int64            `json:"default_retention_seconds"`
	StatusRetentionSeconds  map[string]int64 `json:"status_retention_seconds,omitempty"`
}

func (p NotificationRetentionPolicy) Validate() error {
	if (p.Key != NotificationRetentionHistoryPolicy && p.Key != NotificationRetentionPublicationPolicy) || strings.TrimSpace(p.Version) == "" || p.DefaultRetentionSeconds <= 0 {
		return fmt.Errorf("notification retention policy is invalid")
	}
	for key, seconds := range p.StatusRetentionSeconds {
		if strings.TrimSpace(key) == "" || seconds <= 0 {
			return fmt.Errorf("notification retention status policy is invalid")
		}
	}
	return nil
}

type NotificationRetentionPreviewRequest struct {
	WorkspaceID string                      `json:"workspace_id"`
	Policy      NotificationRetentionPolicy `json:"policy"`
	Now         time.Time                   `json:"now"`
}

type NotificationRetentionPreview struct {
	Rows           int64     `json:"rows"`
	Bytes          int64     `json:"bytes"`
	OldestEligible time.Time `json:"oldest_eligible,omitempty"`
}

type NotificationRetentionHold struct {
	Owner        string     `json:"owner,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	ResourceID   string     `json:"resource_id,omitempty"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
}

type NotificationRetentionBatchRequest struct {
	JobID       string                      `json:"job_id"`
	WorkspaceID string                      `json:"workspace_id"`
	Operation   string                      `json:"operation"`
	DryRun      bool                        `json:"dry_run"`
	Checkpoint  string                      `json:"checkpoint,omitempty"`
	Now         time.Time                   `json:"now"`
	Policy      NotificationRetentionPolicy `json:"policy"`
	Holds       []NotificationRetentionHold `json:"holds,omitempty"`
	Limit       int                         `json:"limit"`
}

type NotificationRetentionBatchResult struct {
	Checkpoint     string    `json:"checkpoint,omitempty"`
	Scanned        int64     `json:"scanned"`
	Archived       int64     `json:"archived"`
	Purged         int64     `json:"purged"`
	Skipped        int64     `json:"skipped"`
	Failed         int64     `json:"failed"`
	OldestEligible time.Time `json:"oldest_eligible,omitempty"`
	Done           bool      `json:"done"`
}

func (r NotificationRetentionPreviewRequest) Validate() error {
	if strings.TrimSpace(r.WorkspaceID) == "" || r.Now.IsZero() {
		return fmt.Errorf("notification retention preview scope is invalid")
	}
	return r.Policy.Validate()
}

func (r NotificationRetentionBatchRequest) Validate() error {
	if strings.TrimSpace(r.JobID) == "" || strings.TrimSpace(r.WorkspaceID) == "" || (r.Operation != "archive" && r.Operation != "purge") || r.Now.IsZero() || r.Limit <= 0 || r.Limit > 1000 {
		return fmt.Errorf("notification retention batch is invalid")
	}
	return r.Policy.Validate()
}
