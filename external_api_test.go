package notificationsdk_test

import (
	"context"
	"testing"

	"github.com/domainry/domainry-foundation/modulecapability"
	notificationsdk "github.com/domainry/domainry-notification-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
)

type externalFactory struct{ binding notificationsdk.Binding }

func (f externalFactory) Open(context.Context, notificationsdk.ApplicationRef) (notificationsdk.Binding, error) {
	return f.binding, nil
}

type externalBinding struct{ modulecapability.Binding }

func (externalBinding) Descriptor() notificationsdk.Descriptor {
	return notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeModule}
}
func (externalBinding) Publisher() notificationsdk.Publisher               { return externalPublisher{} }
func (externalBinding) Inbox() notificationsdk.Inbox                       { return nil }
func (externalBinding) Templates() notificationsdk.Templates               { return nil }
func (externalBinding) Delivery() notificationsdk.Delivery                 { return nil }
func (externalBinding) Administration() notificationsdk.Administration     { return nil }
func (externalBinding) LocalWorkers() (notificationsdk.LocalWorkers, bool) { return nil, false }
func (externalBinding) Close(context.Context) error                        { return nil }

type externalPublisher struct{}

func (externalPublisher) PublishIntent(context.Context, contract.NotificationIntent) (contract.NotificationEvent, bool, error) {
	return contract.NotificationEvent{}, false, nil
}

func TestExternalConsumerCanImplementFactoryAndBinding(t *testing.T) {
	t.Parallel()
	var factory notificationsdk.Factory = externalFactory{binding: externalBinding{}}
	binding, err := factory.Open(t.Context(), notificationsdk.ApplicationRef{})
	if err != nil || binding == nil {
		t.Fatalf("external binding: %v", err)
	}
}
