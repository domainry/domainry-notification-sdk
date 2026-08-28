package modulehost

import (
	"context"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
)

// Factory is the embedded counterpart of notificationsdk.Factory. Runtime
// Host passes one borrowed Host after opening the project database and Identity.
type Factory interface {
	OpenModule(context.Context, notificationsdk.ApplicationRef, Host) (notificationsdk.Binding, error)
}

// TransactionalPublisher is available only when Notification shares the
// caller's database. Remote SaaS Bindings cannot participate in a caller-owned
// SQL transaction and therefore never implement this capability.
type TransactionalPublisher interface {
	CompileIntent(contract.NotificationIntent) (contract.NotificationEvent, error)
	InsertEvent(context.Context, Executor, contract.NotificationEvent) error
	EventCommitted(context.Context, EventIdentity) (bool, error)
}

type EventIdentity struct {
	WorkspaceID   string
	Source        string
	SourceEventID string
}

type TransactionalBinding interface {
	ModuleTransactions() TransactionalPublisher
}
