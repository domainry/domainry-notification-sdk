// Package modulehost defines the host capabilities required only by an
// in-process Notification implementation. Remote Factories never consume this
// package. The interfaces stay implementation-neutral so Runtime depends on
// the SDK rather than Notification internals.
package modulehost

import (
	"context"
	"database/sql"
	"time"

	identitysdk "github.com/domainry/domainry-identity-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
)

type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Database interface {
	Executor
	Queryer
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// SchemaMigration is source-owned DDL applied by the project host through its
// existing migration lock and ledger. BaselineTables allow an installation
// created before module extraction to record the immutable migration without
// re-executing CREATE statements over already-owned tables.
type SchemaMigration struct {
	Version        uint
	Name           string
	Statements     []string
	BaselineTables []string
}

type MigrationRegistrar interface {
	Driver() string
	Schema() string
	ApplyOwnedMigrations(context.Context, string, []SchemaMigration) error
}

type MigrationHost interface {
	Migrations() MigrationRegistrar
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
type RecipientDirectory interface {
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
	Surfaces             []string
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
	RecipientDirectory() RecipientDirectory
	AudienceResolver() AudienceResolver
	DeliveryGateway() DeliveryGateway
	DeliveryMetrics() DeliveryMetrics
	ProviderTemplateValidator() ProviderTemplateValidator
}
