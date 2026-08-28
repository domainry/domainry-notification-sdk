package notificationsdk

import (
	"net/http"
	"testing"
)

func TestServiceGrantForRequestSeparatesPublicationUserAndGovernanceAuthority(t *testing.T) {
	tests := map[string]struct{ resource, action string }{
		"/v1/descriptor":                     {"notification_service", "discover"},
		"/v1/events:publish":                 {"notification_event", "publish"},
		"/v1/inbox:list":                     {"notification_inbox", "read"},
		"/v1/inbox:set-read":                 {"notification_inbox", "update"},
		"/v1/templates:save-draft":           {"notification_template", "draft"},
		"/v1/system/subjects:erase":          {"notification_governance", "erase"},
		"/v1/system/retention:process-batch": {"notification_governance", "retention"},
		"/v1/system/migration:activate":      {"notification_governance", "migrate"},
	}
	for path, want := range tests {
		grant, err := ServiceGrantForRequest(http.MethodPost, path)
		if err != nil || string(grant.Resource) != want.resource || string(grant.Action) != want.action {
			t.Fatalf("path=%s grant=%+v err=%v", path, grant, err)
		}
	}
	if _, err := ServiceGrantForRequest(http.MethodDelete, "/v1/events:publish"); err == nil {
		t.Fatal("unsupported method accepted")
	}
	if _, err := ServiceGrantForRequest(http.MethodPost, "/v1/unknown"); err == nil {
		t.Fatal("unknown route accepted")
	}
}
