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
