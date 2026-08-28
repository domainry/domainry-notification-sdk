package deliverygateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
)

func TestRemoteRetriesUnknownOutcomeWithStableAuthenticatedIdentity(t *testing.T) {
	var mu sync.Mutex
	requests := []Request{}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != AcceptPath || request.Header.Get("X-Domainry-Service-Credential") != "credential" || request.Header.Get("X-Domainry-Tenant-ID") != "tenant" || request.Header.Get("X-Domainry-Workspace-ID") != "workspace" || request.Header.Get("X-Domainry-Application-Key") != "application" {
			http.Error(response, "invalid boundary", http.StatusForbidden)
			return
		}
		var value Request
		if err := json.NewDecoder(request.Body).Decode(&value); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		requests = append(requests, value)
		attempt := len(requests)
		mu.Unlock()
		if attempt == 1 {
			http.Error(response, "unknown outcome", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(response).Encode(Receipt{RequestID: value.RequestID, MessageID: "message"})
	}))
	defer server.Close()
	remote, err := NewRemote(RemoteConfig{BaseURL: server.URL, ServiceCredential: "credential", HTTPClient: server.Client(), RequestTimeout: time.Second, MaxAttempts: 2})
	if err != nil {
		t.Fatal(err)
	}
	application := notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "application"}
	input := Request{RequestID: "request", WorkspaceID: "workspace", PlanID: "plan", EventID: "event", Channel: "email", ConnectorKey: "smtp", Operation: "send", DedupeKey: "dedupe", CreatedAt: "2026-08-28T00:00:00Z"}
	receipt, err := remote.Dispatch(t.Context(), application, input)
	if err != nil || receipt.MessageID != "message" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if len(requests) != 2 || requests[0].RequestID != requests[1].RequestID || requests[0].DedupeKey != requests[1].DedupeKey {
		t.Fatalf("requests=%+v", requests)
	}
}

func TestRequestRequiresExactApplicationWorkspace(t *testing.T) {
	application := notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "application"}
	request := Request{RequestID: "request", WorkspaceID: "other", PlanID: "plan", EventID: "event", Channel: "email", ConnectorKey: "smtp", Operation: "send", DedupeKey: "dedupe", CreatedAt: "now"}
	if err := request.Validate(application); err == nil {
		t.Fatal("cross-workspace Delivery Gateway request was accepted")
	}
}
