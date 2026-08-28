package contract

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const NotificationPortableFormatV1 = "domainry-notification-portable-v1"

type NotificationPortableScope struct {
	TenantID       string `json:"tenant_id"`
	WorkspaceID    string `json:"workspace_id"`
	ApplicationKey string `json:"application_key"`
}

func (s NotificationPortableScope) Validate() error {
	if strings.TrimSpace(s.TenantID) == "" || strings.TrimSpace(s.WorkspaceID) == "" || strings.TrimSpace(s.ApplicationKey) == "" {
		return fmt.Errorf("notification portable scope is invalid")
	}
	return nil
}

type NotificationPortableTable struct {
	Name    string              `json:"name"`
	Columns []string            `json:"columns"`
	Rows    [][]json.RawMessage `json:"rows"`
}

type NotificationPortableBundle struct {
	FormatVersion string                      `json:"format_version"`
	Source        NotificationPortableScope   `json:"source"`
	Tables        []NotificationPortableTable `json:"tables"`
	Fingerprint   string                      `json:"fingerprint"`
}

func (b NotificationPortableBundle) ValidateEnvelope() error {
	if b.FormatVersion != NotificationPortableFormatV1 || b.Source.Validate() != nil || strings.TrimSpace(b.Fingerprint) == "" || len(b.Tables) == 0 {
		return fmt.Errorf("notification portable bundle envelope is invalid")
	}
	return nil
}

type NotificationPortableInventory struct {
	Tables       map[string]int `json:"tables"`
	Rows         int            `json:"rows"`
	ActiveLeases int            `json:"active_leases"`
	Fingerprint  string         `json:"fingerprint"`
}

type NotificationPortableExport struct {
	Bundle    NotificationPortableBundle    `json:"bundle"`
	Inventory NotificationPortableInventory `json:"inventory"`
}

type NotificationPortableImportReceipt struct {
	FormatVersion  string `json:"format_version"`
	Fingerprint    string `json:"fingerprint"`
	Rows           int    `json:"rows"`
	AlreadyPresent bool   `json:"already_present"`
}

type NotificationMigrationState string

const (
	NotificationMigrationActive   NotificationMigrationState = "active"
	NotificationMigrationFrozen   NotificationMigrationState = "frozen"
	NotificationMigrationImported NotificationMigrationState = "imported"
	NotificationMigrationCutover  NotificationMigrationState = "cutover"
)

type NotificationMigrationCommand struct {
	MigrationID       string    `json:"migration_id"`
	BundleFingerprint string    `json:"bundle_fingerprint,omitempty"`
	At                time.Time `json:"at"`
}

func (c NotificationMigrationCommand) Validate(requireFingerprint bool) error {
	if strings.TrimSpace(c.MigrationID) == "" || c.At.IsZero() || requireFingerprint && strings.TrimSpace(c.BundleFingerprint) == "" {
		return fmt.Errorf("notification migration command is invalid")
	}
	return nil
}

type NotificationMigrationStatus struct {
	MigrationID       string                     `json:"migration_id,omitempty"`
	Role              string                     `json:"role,omitempty"`
	State             NotificationMigrationState `json:"state"`
	BundleFingerprint string                     `json:"bundle_fingerprint,omitempty"`
	FrozenAt          time.Time                  `json:"frozen_at,omitempty"`
	ActivatedAt       time.Time                  `json:"activated_at,omitempty"`
	ActiveLeases      int                        `json:"active_leases"`
}
