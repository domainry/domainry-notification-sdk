package remote_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
	"github.com/domainry/domainry-notification-sdk/contracttest"
	"github.com/domainry/domainry-notification-sdk/remote"
)

func TestRemoteFactoryContract(t *testing.T) {
	server := httptest.NewServer(remoteTestHandler(t, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/notification/v1/descriptor" {
			http.NotFound(response, request)
			return
		}
		_ = json.NewEncoder(response).Encode(notificationsdk.Descriptor{ProtocolVersion: notificationsdk.CurrentProtocolVersion, Mode: notificationsdk.DeploymentModeSaaS, Audience: "runtime"})
	})))
	defer server.Close()
	contracttest.Run(t, func(testing.TB) (notificationsdk.Factory, notificationsdk.ApplicationRef) {
		return remote.NewFactory(remote.Config{BaseURL: server.URL, ServiceCredential: "service", CapabilityContractSHA256: remoteTestCapabilitySHA256(t), HTTPClient: server.Client()}), notificationsdk.ApplicationRef{TenantID: "tenant", WorkspaceID: "workspace", ApplicationKey: "runtime"}
	})
}
