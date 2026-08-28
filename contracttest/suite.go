// Package contracttest contains one deployment-neutral acceptance suite that
// every Notification Factory must pass.
package contracttest

import (
	"context"
	"testing"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
)

type FactoryProvider func(testing.TB) (notificationsdk.Factory, notificationsdk.ApplicationRef)

func Run(t *testing.T, provide FactoryProvider) {
	t.Helper()
	t.Run("opens complete scoped binding", func(t *testing.T) {
		factory, application := provide(t)
		if factory == nil {
			t.Fatal("Factory is nil")
		}
		binding, err := factory.Open(t.Context(), application)
		if err != nil {
			t.Fatalf("open Binding: %v", err)
		}
		if binding == nil || binding.Publisher() == nil || binding.Inbox() == nil || binding.Templates() == nil || binding.Delivery() == nil || binding.Administration() == nil {
			t.Fatal("Binding returned incomplete use-case ports")
		}
		if err := binding.Descriptor().Validate(); err != nil {
			t.Fatalf("invalid descriptor: %v", err)
		}
		workers, local := binding.LocalWorkers()
		if binding.Descriptor().Mode == notificationsdk.DeploymentModeModule && (!local || workers == nil) {
			t.Fatal("Module Binding returned no local worker capability")
		}
		if binding.Descriptor().Mode == notificationsdk.DeploymentModeSaaS && (local || workers != nil) {
			t.Fatal("SaaS Binding exposed local workers")
		}
		if err := binding.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Fatalf("close Binding: %v", err)
		}
	})
	t.Run("rejects incomplete application scope", func(t *testing.T) {
		factory, _ := provide(t)
		if _, err := factory.Open(t.Context(), notificationsdk.ApplicationRef{}); err == nil {
			t.Fatal("Factory accepted incomplete application scope")
		}
	})
}
