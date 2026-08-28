package notificationsdk_test

import (
	"encoding/json"
	"testing"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
)

func TestApplicationRefRequiresExplicitApplicationScope(t *testing.T) {
	t.Parallel()
	for _, ref := range []notificationsdk.ApplicationRef{
		{},
		{TenantID: "tenant", WorkspaceID: "workspace"},
		{TenantID: "tenant", ApplicationKey: "notification"},
		{WorkspaceID: "workspace", ApplicationKey: "notification"},
	} {
		if ref.Validate() == nil {
			t.Fatalf("expected invalid application ref: %#v", ref)
		}
	}
	if err := (notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "notification"}).Validate(); err != nil {
		t.Fatalf("validate complete application ref: %v", err)
	}
}

func TestErrorWireContractOmitsLocalState(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(&notificationsdk.Error{StatusCode: 503, Code: "notification.remote_unavailable", Message: "temporarily unavailable", Retryable: true, Cause: assertError("secret local cause")})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"code":"notification.remote_unavailable","message":"temporarily unavailable","retryable":true}`
	if string(encoded) != want {
		t.Fatalf("error wire drift:\n got %s\nwant %s", encoded, want)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }

func TestDescriptorRejectsUnknownProtocolAndMode(t *testing.T) {
	t.Parallel()
	if err := (notificationsdk.Descriptor{Mode: notificationsdk.DeploymentModeModule}).Validate(); err == nil {
		t.Fatal("expected missing protocol to fail")
	}
	if err := (notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: "legacy"}).Validate(); err == nil {
		t.Fatal("expected legacy mode to fail")
	}
	if err := (notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS}).Validate(); err != nil {
		t.Fatalf("validate SaaS descriptor: %v", err)
	}
}
