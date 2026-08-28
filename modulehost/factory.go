package modulehost

import (
	"context"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
)

// Factory is the embedded counterpart of notificationsdk.Factory. Runtime
// Host passes one borrowed Host after opening the project database and Identity.
type Factory interface {
	OpenModule(context.Context, notificationsdk.ApplicationRef, Host) (notificationsdk.Binding, error)
}
