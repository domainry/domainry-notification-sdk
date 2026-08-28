// Package notificationsdk composes deployment-neutral Notification contracts.
package notificationsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/domainry/domainry-notification-sdk/contract"
)

type DeploymentMode string

const (
	DeploymentModeModule   DeploymentMode = "module"
	DeploymentModeSaaS     DeploymentMode = "saas"
	ProtocolVersionV1                     = "domainry-notification-protocol-v1"
	CurrentProtocolVersion                = ProtocolVersionV1
)

// Error is stable across in-process and Remote implementations. Cause is
// local-only and is never serialized.
type Error struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message,omitempty"`
	Retryable  bool   `json:"retryable,omitempty"`
	Cause      error  `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// ApplicationRef binds one Factory instance to an exact scope. No field may
// be inferred from another field.
type ApplicationRef struct {
	TenantID       string `json:"tenant_id"`
	WorkspaceID    string `json:"workspace_id"`
	ApplicationKey string `json:"application_key"`
}

func (r ApplicationRef) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" || strings.TrimSpace(r.WorkspaceID) == "" || strings.TrimSpace(r.ApplicationKey) == "" {
		return &Error{StatusCode: 400, Code: "notification.application_scope_invalid"}
	}
	return nil
}

type Descriptor struct {
	ProtocolVersion string         `json:"protocol_version"`
	Mode            DeploymentMode `json:"mode"`
	Audience        string         `json:"audience,omitempty"`
	Capabilities    []string       `json:"capabilities"`
}

func (d Descriptor) Validate() error {
	if d.ProtocolVersion != CurrentProtocolVersion {
		return &Error{StatusCode: 502, Code: "notification.protocol_version_unsupported", Message: fmt.Sprintf("got %q", d.ProtocolVersion)}
	}
	if d.Mode != DeploymentModeModule && d.Mode != DeploymentModeSaaS {
		return &Error{StatusCode: 502, Code: "notification.deployment_mode_invalid"}
	}
	return nil
}

type Factory interface {
	Open(context.Context, ApplicationRef) (Binding, error)
}

// DatabaseHandle is borrowed from the project host. An in-process Factory must
// never close Pool; a Remote Factory does not implement DatabaseFactory.
type DatabaseHandle struct {
	Pool                     any
	Driver, Schema, FilePath string
}
type DatabaseFactory interface {
	OpenWithDatabase(context.Context, ApplicationRef, DatabaseHandle) (Binding, error)
}

// Binding exposes business use cases, never stores, processors or worker loops.
type Binding interface {
	Descriptor() Descriptor
	Publisher() Publisher
	Inbox() Inbox
	Templates() Templates
	Delivery() Delivery
	Administration() Administration
	LocalWorkers() (LocalWorkers, bool)
	Close(context.Context) error
}

// SystemTemplates is the service-authenticated installation boundary used by
// a Runtime host during metadata restoration. It is deliberately separate
// from user-authorized template administration.
type SystemTemplates interface {
	SyncPublished(context.Context, []contract.NotificationTemplate) error
	ListPublished(context.Context) ([]contract.NotificationTemplateRecord, error)
}

type SystemTemplateBinding interface {
	SystemTemplates() SystemTemplates
}

type SystemSubjects interface {
	PreviewSubject(context.Context, string, string) (json.RawMessage, error)
	ExportSubject(context.Context, string, string) (json.RawMessage, error)
	EraseSubject(context.Context, string, string, json.RawMessage) (json.RawMessage, error)
}

type SystemSubjectBinding interface {
	SystemSubjects() SystemSubjects
}

type SystemRetention interface {
	Preview(context.Context, contract.NotificationRetentionPreviewRequest) (contract.NotificationRetentionPreview, error)
	ProcessBatch(context.Context, contract.NotificationRetentionBatchRequest) (contract.NotificationRetentionBatchResult, error)
}

type SystemRetentionBinding interface {
	SystemRetention() SystemRetention
}

// SystemMigration is a service-authenticated, application-scoped portable
// state boundary. Export includes a deterministic inventory and fingerprint;
// Import must be idempotent and reject any non-matching existing state.
type SystemMigration interface {
	Status(context.Context) (contract.NotificationMigrationStatus, error)
	Freeze(context.Context, contract.NotificationMigrationCommand) (contract.NotificationMigrationStatus, error)
	Export(context.Context) (contract.NotificationPortableExport, error)
	Import(context.Context, contract.NotificationPortableBundle) (contract.NotificationPortableImportReceipt, error)
	Activate(context.Context, contract.NotificationMigrationCommand) (contract.NotificationMigrationStatus, error)
	Rollback(context.Context, contract.NotificationMigrationCommand) (contract.NotificationMigrationStatus, error)
}

type SystemMigrationBinding interface {
	SystemMigration() SystemMigration
}

type WorkLocator struct {
	Kind        string `json:"kind"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	TaskID      string `json:"task_id"`
}

// LocalWorkers is available only from an embedded Module Binding. SaaS owns
// its process workers and Remote Bindings return (_, false).
type LocalWorkers interface {
	ProcessDuePublications(context.Context, int) (int, error)
	ProcessPublication(context.Context, WorkLocator) (bool, error)
	RefreshPublished(context.Context) error
	ProcessDueInboxEvents(context.Context, int) (int, error)
	ProcessInboxEvent(context.Context, WorkLocator) (bool, error)
	ProcessDueChannelPlans(context.Context, int) (int, error)
	ProcessChannelPlan(context.Context, WorkLocator) (bool, error)
}

// UserAuthority carries the original Identity token. Module and SaaS both
// verify it and never trust caller-supplied user IDs or permission lists.
type UserAuthority struct {
	AccessToken string `json:"-"`
	Surface     string `json:"-"`
}

func (a UserAuthority) Validate() error {
	if strings.TrimSpace(a.AccessToken) == "" {
		return &Error{StatusCode: 401, Code: "notification.user_authority_required"}
	}
	return nil
}

type Publisher interface {
	PublishIntent(context.Context, contract.NotificationIntent) (contract.NotificationEvent, bool, error)
}

type Inbox interface {
	List(context.Context, UserAuthority, contract.NotificationInboxQuery, string) (contract.NotificationInboxPage, error)
	Get(context.Context, UserAuthority, string, contract.NotificationInboxQuery) (contract.NotificationInboxItem, error)
	Facets(context.Context, UserAuthority, contract.NotificationInboxQuery) (contract.NotificationInboxFacets, error)
	SetRead(context.Context, UserAuthority, string, bool) (contract.NotificationInboxItem, error)
	SetArchived(context.Context, UserAuthority, string, bool) (contract.NotificationInboxItem, error)
	AcknowledgeAlert(context.Context, UserAuthority, string) (contract.NotificationInboxItem, error)
	MarkAllRead(context.Context, UserAuthority, contract.NotificationInboxQuery) (int, error)
	ResolveAction(context.Context, UserAuthority, string, string, contract.NotificationInboxQuery) (contract.NotificationInboxResolvedAction, error)
	ListDelegations(context.Context, UserAuthority, string) ([]contract.NotificationInboxDelegation, error)
	SaveDelegation(context.Context, UserAuthority, contract.NotificationInboxDelegation) (contract.NotificationInboxDelegation, error)
	DeleteDelegation(context.Context, UserAuthority, string) error
	ListDelegatedOwnerIDs(context.Context, UserAuthority, string) ([]string, error)
	ListSavedViews(context.Context, UserAuthority, string) ([]contract.NotificationInboxSavedView, error)
	SaveSavedView(context.Context, UserAuthority, contract.NotificationInboxSavedView) (contract.NotificationInboxSavedView, error)
	DeleteSavedView(context.Context, UserAuthority, string) error
	GetPreference(context.Context, UserAuthority, string) (contract.NotificationRecipientPreference, error)
	SavePreference(context.Context, UserAuthority, string, contract.NotificationRecipientPreference) (contract.NotificationRecipientPreference, error)
}

type Templates interface {
	Capabilities(context.Context, UserAuthority) ([]contract.NotificationTemplateCapability, error)
	List(context.Context, UserAuthority) ([]contract.NotificationTemplateRecord, error)
	Get(context.Context, UserAuthority, string) (contract.NotificationTemplateRecord, bool, error)
	ListVersions(context.Context, UserAuthority, string) ([]contract.NotificationTemplateVersion, error)
	SaveDraft(context.Context, UserAuthority, string, contract.NotificationTemplate, string) (contract.NotificationTemplateRecord, error)
	RestoreVersionDraft(context.Context, UserAuthority, string, int, string) (contract.NotificationTemplateRecord, error)
	Disable(context.Context, UserAuthority, string, string) (contract.NotificationTemplateRecord, error)
	Preview(context.Context, UserAuthority, string, string, []string, map[string]any) (contract.RenderedNotification, error)
	PreviewTemplate(context.Context, UserAuthority, contract.NotificationTemplate, string, []string, map[string]any) (contract.RenderedNotification, error)
	ListPublicationRequests(context.Context, UserAuthority, string) ([]contract.NotificationPublicationRequest, error)
	RequestPublication(context.Context, UserAuthority, string, string, string) (contract.NotificationPublicationRequest, error)
	ApprovePublication(context.Context, UserAuthority, string) (contract.NotificationPublicationRequest, error)
	RejectPublication(context.Context, UserAuthority, string, string) (contract.NotificationPublicationRequest, error)
	CancelPublication(context.Context, UserAuthority, string) (contract.NotificationPublicationRequest, error)
}

type Delivery interface {
	GetPolicy(context.Context, UserAuthority) (contract.NotificationDeliveryPolicy, error)
	SavePolicy(context.Context, UserAuthority, contract.NotificationDeliveryPolicy) (contract.NotificationDeliveryPolicy, error)
	ListRecipientPreferences(context.Context, UserAuthority) ([]contract.NotificationRecipientPreference, error)
	SaveRecipientPreference(context.Context, UserAuthority, contract.NotificationRecipientPreference) (contract.NotificationRecipientPreference, error)
	Metrics(context.Context, UserAuthority, string) (contract.NotificationDeliveryMetrics, error)
}

type Administration interface {
	GovernanceCatalog(context.Context, UserAuthority) (contract.NotificationGovernanceCatalog, error)
	InboxGovernanceMetrics(context.Context, UserAuthority, string) (contract.NotificationInboxGovernanceMetrics, error)
}
