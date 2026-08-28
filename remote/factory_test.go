package remote_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
	"github.com/domainry/domainry-notification-sdk/remote"
)

func TestFactoryDiscoversSaaSAndBindsExactApplicationHeaders(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/descriptor":
			if request.Header.Get("X-Domainry-Service-Credential") != "service-secret" || request.Header.Get("X-Domainry-Tenant-ID") != "tenant" || request.Header.Get("X-Domainry-Workspace-ID") != "workspace" || request.Header.Get("X-Domainry-Application-Key") != "runtime" {
				t.Errorf("discovery scope headers are incomplete: %#v", request.Header)
			}
			_ = json.NewEncoder(response).Encode(notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "runtime"})
		case "/v1/events:publish":
			if request.Header.Get("Authorization") != "" {
				t.Error("service publication unexpectedly carried user bearer authority")
			}
			_ = json.NewEncoder(response).Encode(map[string]any{"event": contract.NotificationEvent{ID: "event"}, "created": true})
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	factory := remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "service-secret", HTTPClient: server.Client()})
	binding, err := factory.Open(t.Context(), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"})
	if err != nil {
		t.Fatal(err)
	}
	intent := contract.NotificationIntent{ID: "event", WorkspaceID: "workspace", SourceEventID: "source", EventType: "report.completed", Surface: "business_workspace", RecipientUserIDs: []string{"user"}, OccurredAt: "2026-08-28T00:00:00.000000000Z", SubjectType: "report", SubjectID: "report", SubjectVersion: "one"}
	event, created, err := binding.Publisher().PublishIntent(t.Context(), intent)
	if err != nil || !created || event.ID != "event" {
		t.Fatalf("publish result: event=%#v created=%t err=%v", event, created, err)
	}
}

func TestRemoteSystemTemplatesUseOnlyServiceAuthority(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "" || request.Header.Get("X-Domainry-Service-Credential") != "service-secret" {
			t.Fatalf("unexpected system authority headers: %#v", request.Header)
		}
		switch request.URL.Path {
		case "/v1/descriptor":
			_ = json.NewEncoder(response).Encode(notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "runtime"})
		case "/v1/system/templates:sync-published":
			response.WriteHeader(http.StatusNoContent)
		case "/v1/system/templates:list-published":
			_ = json.NewEncoder(response).Encode([]contract.NotificationTemplateRecord{{Key: "welcome"}})
		case "/v1/system/subjects:preview", "/v1/system/subjects:export", "/v1/system/subjects:erase":
			_ = json.NewEncoder(response).Encode(map[string]any{"subject": "user"})
		case "/v1/system/retention:preview":
			_ = json.NewEncoder(response).Encode(contract.NotificationRetentionPreview{Rows: 3})
		case "/v1/system/retention:process-batch":
			_ = json.NewEncoder(response).Encode(contract.NotificationRetentionBatchResult{Scanned: 2, Purged: 1, Done: true})
		case "/v1/system/migration:export":
			_ = json.NewEncoder(response).Encode(contract.NotificationPortableExport{Bundle: portableBundle(), Inventory: contract.NotificationPortableInventory{Rows: 1, Fingerprint: "fingerprint"}})
		case "/v1/system/migration:import":
			_ = json.NewEncoder(response).Encode(contract.NotificationPortableImportReceipt{FormatVersion: contract.NotificationPortableFormatV1, Fingerprint: "fingerprint", Rows: 1})
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	binding, err := remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "service-secret", HTTPClient: server.Client()}).Open(t.Context(), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"})
	if err != nil {
		t.Fatal(err)
	}
	system, ok := binding.(notificationsdk.SystemTemplateBinding)
	if !ok || system.SystemTemplates() == nil {
		t.Fatal("Remote Binding did not expose system templates")
	}
	if err := system.SystemTemplates().SyncPublished(t.Context(), []contract.NotificationTemplate{{Key: "welcome"}}); err != nil {
		t.Fatal(err)
	}
	records, err := system.SystemTemplates().ListPublished(t.Context())
	if err != nil || len(records) != 1 || records[0].Key != "welcome" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	subjects, ok := binding.(notificationsdk.SystemSubjectBinding)
	if !ok || subjects.SystemSubjects() == nil {
		t.Fatal("Remote Binding did not expose system subjects")
	}
	for _, call := range []func() (json.RawMessage, error){
		func() (json.RawMessage, error) {
			return subjects.SystemSubjects().PreviewSubject(t.Context(), "workspace", "user")
		},
		func() (json.RawMessage, error) {
			return subjects.SystemSubjects().ExportSubject(t.Context(), "workspace", "user")
		},
		func() (json.RawMessage, error) {
			return subjects.SystemSubjects().EraseSubject(t.Context(), "workspace", "user", json.RawMessage(`[]`))
		},
	} {
		value, callErr := call()
		if callErr != nil || !bytes.Contains(value, []byte(`"subject":"user"`)) {
			t.Fatalf("subject value=%s err=%v", value, callErr)
		}
	}
	retentionBinding, ok := binding.(notificationsdk.SystemRetentionBinding)
	if !ok || retentionBinding.SystemRetention() == nil {
		t.Fatal("Remote Binding did not expose system retention")
	}
	policy := contract.NotificationRetentionPolicy{Key: contract.NotificationRetentionHistoryPolicy, Version: "1", DefaultRetentionSeconds: 3600}
	now := time.Date(2026, 8, 29, 1, 0, 0, 0, time.UTC)
	preview, err := retentionBinding.SystemRetention().Preview(t.Context(), contract.NotificationRetentionPreviewRequest{WorkspaceID: "workspace", Policy: policy, Now: now})
	if err != nil || preview.Rows != 3 {
		t.Fatalf("retention preview=%+v err=%v", preview, err)
	}
	batch, err := retentionBinding.SystemRetention().ProcessBatch(t.Context(), contract.NotificationRetentionBatchRequest{JobID: "job", WorkspaceID: "workspace", Operation: "purge", Policy: policy, Now: now, Limit: 10})
	if err != nil || batch.Purged != 1 || !batch.Done {
		t.Fatalf("retention batch=%+v err=%v", batch, err)
	}
	migrationBinding, ok := binding.(notificationsdk.SystemMigrationBinding)
	if !ok || migrationBinding.SystemMigration() == nil {
		t.Fatal("Remote Binding did not expose system migration")
	}
	exported, err := migrationBinding.SystemMigration().Export(t.Context())
	if err != nil || exported.Bundle.Fingerprint != "fingerprint" || exported.Inventory.Rows != 1 {
		t.Fatalf("migration export=%+v err=%v", exported, err)
	}
	receipt, err := migrationBinding.SystemMigration().Import(t.Context(), exported.Bundle)
	if err != nil || receipt.Fingerprint != exported.Bundle.Fingerprint || receipt.Rows != 1 {
		t.Fatalf("migration receipt=%+v err=%v", receipt, err)
	}
}

func portableBundle() contract.NotificationPortableBundle {
	return contract.NotificationPortableBundle{
		FormatVersion: contract.NotificationPortableFormatV1,
		Source:        contract.NotificationPortableScope{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"},
		Tables:        []contract.NotificationPortableTable{{Name: "notification_events", Columns: []string{"id"}, Rows: [][]json.RawMessage{{json.RawMessage(`"event"`)}}}},
		Fingerprint:   "fingerprint",
	}
}

func TestRemoteRetriesStablePublicationButDoesNotReplayOrdinaryMutation(t *testing.T) {
	t.Parallel()
	discovery, publications, mutations := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/descriptor":
			discovery++
			_ = json.NewEncoder(response).Encode(notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "runtime"})
		case "/v1/events:publish":
			publications++
			if publications == 1 {
				response.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(response).Encode(notificationsdk.Error{Code: "notification.unavailable", Retryable: true})
				return
			}
			_ = json.NewEncoder(response).Encode(map[string]any{"event": contract.NotificationEvent{ID: "event"}, "created": true})
		case "/v1/inbox:set-read":
			mutations++
			response.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(response).Encode(notificationsdk.Error{Code: "notification.unavailable", Retryable: true})
		}
	}))
	defer server.Close()
	factory := remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "service", HTTPClient: server.Client(), Retry: remote.RetryPolicy{MaxAttempts: 2, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond}})
	binding, err := factory.Open(t.Context(), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"})
	if err != nil {
		t.Fatal(err)
	}
	intent := contract.NotificationIntent{ID: "event", WorkspaceID: "workspace", SourceEventID: "source", EventType: "report.completed", Surface: "business_workspace", RecipientUserIDs: []string{"user"}, OccurredAt: "2026-08-28T00:00:00.000000000Z", SubjectType: "report", SubjectID: "report", SubjectVersion: "one"}
	if _, _, err := binding.Publisher().PublishIntent(t.Context(), intent); err != nil {
		t.Fatal(err)
	}
	if _, err := binding.Inbox().SetRead(t.Context(), notificationsdk.UserAuthority{AccessToken: "token"}, "item", true); err == nil {
		t.Fatal("expected mutation failure")
	}
	if discovery != 1 || publications != 2 || mutations != 1 {
		t.Fatalf("unexpected attempts discovery=%d publication=%d mutation=%d", discovery, publications, mutations)
	}
}

func TestRemotePropagatesTraceHeadersWithoutAllowingCredentialOverride(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Traceparent") != "00-trace-span-01" {
			t.Errorf("trace header missing: %#v", request.Header)
		}
		if request.Header.Get("X-Domainry-Service-Credential") != "real-service" {
			t.Errorf("protected credential was overwritten")
		}
		_ = json.NewEncoder(response).Encode(notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "runtime"})
	}))
	defer server.Close()
	factory := remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "real-service", HTTPClient: server.Client(), ContextHeaders: func(context.Context) http.Header {
		return http.Header{"Traceparent": []string{"00-trace-span-01"}, "X-Domainry-Service-Credential": []string{"evil"}}
	}})
	if _, err := factory.Open(t.Context(), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"}); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryRejectsModuleDescriptorAndAudienceMismatch(t *testing.T) {
	t.Parallel()
	for _, descriptor := range []notificationsdk.Descriptor{
		{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeModule, Audience: "runtime"},
		{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "different"},
	} {
		descriptor := descriptor
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(response).Encode(descriptor) }))
		factory := remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "service-secret", HTTPClient: server.Client()})
		if _, err := factory.Open(t.Context(), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"}); err == nil {
			t.Fatalf("accepted invalid descriptor %#v", descriptor)
		}
		server.Close()
	}
}

func TestUserUseCaseRequiresAndForwardsIdentityBearerToken(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/descriptor" {
			_ = json.NewEncoder(response).Encode(notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "runtime"})
			return
		}
		if request.Header.Get("Authorization") != "Bearer identity-token" {
			t.Errorf("Identity token was not forwarded: %q", request.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(response).Encode(contract.NotificationInboxPage{})
	}))
	defer server.Close()
	binding, err := remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "service-secret", HTTPClient: server.Client()}).Open(t.Context(), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := binding.Inbox().List(t.Context(), notificationsdk.UserAuthority{}, contract.NotificationInboxQuery{}, ""); err == nil {
		t.Fatal("accepted missing Identity authority")
	}
	if _, err := binding.Inbox().List(t.Context(), notificationsdk.UserAuthority{AccessToken: "identity-token"}, contract.NotificationInboxQuery{}, ""); err != nil {
		t.Fatal(err)
	}
}
