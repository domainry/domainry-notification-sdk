// Package modulehost defines the host capabilities required only by an
// in-process Notification implementation. Remote Factories never consume this
// package. The interfaces stay implementation-neutral so Runtime depends on
// the SDK rather than Notification internals.
package modulehost

import (
	"context"
	"time"

	identitysdk "github.com/domainry/domainry-identity-sdk"
	metadatasdk "github.com/domainry/domainry-metadata-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
	ormmigration "github.com/domainry/domainry-orm/migration"
	"github.com/domainry/domainry-orm/sqlhost"
)

type Executor = sqlhost.Executor
type Queryer = sqlhost.Queryer
type Database = sqlhost.Database

// SchemaBaseline is the exact source-owned physical shape that a host must
// prove before adopting a schema created before module extraction. Indexes are
// the source-declared indexes; database-generated primary-key indexes are not
// part of this inventory.
type SchemaBaseline = ormmigration.Baseline
type SchemaTable = ormmigration.Table
type SchemaColumn = ormmigration.Column
type SchemaIndex = ormmigration.Index

// SchemaMigration is source-owned DDL applied by the project host through its
// existing migration lock and ledger. Baseline allows an installation created
// before module extraction to record the immutable migration only after the
// host proves every declared table, column, physical type, nullability,
// primary key and explicit index.
type SchemaMigration = ormmigration.Migration

type MigrationRegistrar interface {
	Driver() string
	Schema() string
	ApplyOwnedMigrations(context.Context, string, []SchemaMigration) error
}

type MigrationHost interface {
	Migrations() MigrationRegistrar
}

// DefinitionStoreHost supplies the installation-wide versioned Definition
// store used by Notification policies and templates. Notification must not
// create private definition/version tables beside this shared catalog.
type DefinitionStoreHost interface {
	DefinitionStore() metadatasdk.DefinitionStore
}

// ManagedOperationStoreHost supplies the installation-wide Operation ledger
// used by template publication requests and their fenced worker lease.
type ManagedOperationStoreHost interface {
	ManagedOperationStore() ManagedOperationStore
}

// OperationControlStoreHost supplies the installation-wide durable control
// registry used to fence Notification data cutover across Runtime instances.
type OperationControlStoreHost interface {
	OperationControlStore() OperationControlStore
}

type RetentionArchiveStoreHost interface {
	RetentionArchiveStore() RetentionArchiveStore
}

type Dialect interface {
	Identifier(string) string
	Table(string) string
	Placeholder(int) string
	Insert(string, []string) string
}

type WorkspaceScope interface {
	Context(context.Context, string) context.Context
}
type QueueScopeIndex interface {
	Register(context.Context, Executor, string, string, string) error
	Workspaces(context.Context, Queryer, string, int) ([]string, error)
}

type Clock interface{ Now() time.Time }
type WorkLocator struct{ Kind, WorkspaceID, TaskID string }
type WorkNotifier interface {
	Notify(context.Context, WorkLocator)
}

type Recipient struct{ ID, Email, Locale, Timezone string }
type RecipientResolver interface {
	FindRecipient(context.Context, string, string) (Recipient, bool, error)
}

// AudienceResolver handles only Runtime-business audience facts. Identity
// users, membership and organization facts are resolved through Identity.
type AudienceResolver interface {
	ResolveAudience(context.Context, string, contract.NotificationEvent) ([]string, error)
}

type DeliveryRequest struct {
	WorkspaceID, PlanID, EventID, Channel, ConnectorKey, ConnectionKey, Operation, DedupeKey string
	DeliverAfter, CreatedAt                                                                  string
	Rendered                                                                                 contract.RenderedNotification
	Fallbacks                                                                                []DeliveryFallback
}
type DeliveryFallback struct {
	ConnectorKey, ConnectionKey, Operation string
	Rendered                               contract.RenderedNotification
}
type DeliveryReceipt struct{ MessageID string }
type DeliveryGateway interface {
	Dispatch(context.Context, DeliveryRequest) (DeliveryReceipt, error)
}

type DeliveryMetrics interface {
	Metrics(context.Context, string, string) (contract.NotificationDeliveryMetrics, error)
}

type ProviderTemplateValidator interface {
	ValidateProviderTemplate(string, string, contract.NotificationProviderTemplate) error
}

type Catalog struct {
	DefaultLocale        string
	ExternalChannels     []string
	Templates            []contract.NotificationTemplate
	TemplateCapabilities []contract.NotificationTemplateCapability
	EventTypes           []contract.NotificationEventType
	Rules                []contract.NotificationRule
}

// Host is one already-opened project composition. Database and Identity are
// borrowed; Notification must close neither of them.
type Host interface {
	Database() Database
	Dialect() Dialect
	WorkspaceScope() WorkspaceScope
	QueueScopes() QueueScopeIndex
	Identity() identitysdk.Binding
	Clock() Clock
	WorkerID() string
	Catalog() Catalog
	WorkNotifier() WorkNotifier
	RecipientResolver() RecipientResolver
	AudienceResolver() AudienceResolver
	DeliveryGateway() DeliveryGateway
	DeliveryMetrics() DeliveryMetrics
	ProviderTemplateValidator() ProviderTemplateValidator
}
