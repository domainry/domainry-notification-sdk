package contract_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/domainry/domainry-notification-sdk/contract"
)

func TestNotificationIntentWireContractIsDeterministic(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(contract.NotificationIntent{
		ID: "event-1", WorkspaceID: "workspace-1", SourceEventID: "source-1", EventType: "report.completed",
		RecipientUserIDs: []string{"user-1"}, SubjectType: "report", SubjectID: "report-1",
		SubjectVersion: "version-1", OccurredAt: "2026-08-28T00:00:00.000000000Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"event-1","workspace_id":"workspace-1","source_event_id":"source-1","event_type":"report.completed","recipient_user_ids":["user-1"],"subject_type":"report","subject_id":"report-1","subject_version":"version-1","occurred_at":"2026-08-28T00:00:00.000000000Z"}`
	if string(encoded) != want {
		t.Fatalf("intent wire drift:\n got %s\nwant %s", encoded, want)
	}
}

func TestInboxQueryWireExcludesServerDerivedAuthority(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(contract.NotificationInboxQuery{Scope: contract.NotificationInboxScopeTeam, TeamMemberID: "user-2", Mailbox: contract.NotificationMailboxUnread, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"scope":"team","mailbox":"unread","team_member_id":"user-2","limit":50}`
	if string(encoded) != want {
		t.Fatalf("query wire drift:\n got %s\nwant %s", encoded, want)
	}
	for _, forbidden := range []string{"workspace_id", "viewer_user_id", "reporting_user_ids", "delegated_user_ids", "recipient_user_ids"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("query exposed server-derived authority %q: %s", forbidden, encoded)
		}
	}
}
