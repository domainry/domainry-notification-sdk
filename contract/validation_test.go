package contract_test

import (
	"errors"
	"testing"

	"github.com/domainry/domainry-notification-sdk/contract"
)

func validIntent() contract.NotificationIntent {
	return contract.NotificationIntent{
		ID: "event", WorkspaceID: "workspace", SourceEventID: "source-event", EventType: "report.completed",
		Surface: "business_workspace", RecipientUserIDs: []string{"user"}, OccurredAt: "2026-08-28T00:00:00.000000000Z",
		SubjectType: "report", SubjectID: "report-1", SubjectVersion: "version-1",
	}
}

func TestNotificationIntentRequiresVersionedSubjectAndBoundedAudience(t *testing.T) {
	t.Parallel()
	value := validIntent()
	if err := value.Validate(); err != nil {
		t.Fatalf("validate intent: %v", err)
	}
	value.SubjectVersion = ""
	if !errors.Is(value.Validate(), contract.ErrIntentSubjectRequired) {
		t.Fatalf("expected subject-version failure, got %v", value.Validate())
	}
	value = validIntent()
	value.RecipientUserIDs = make([]string, contract.MaximumExplicitRecipients+1)
	if !errors.Is(value.Validate(), contract.ErrIntentAudienceTooLarge) {
		t.Fatalf("expected audience bound failure, got %v", value.Validate())
	}
}

func TestInboxQueryRejectsCallerSuppliedInvalidScopeAndUnboundedPage(t *testing.T) {
	t.Parallel()
	if err := (contract.NotificationInboxQuery{Scope: contract.NotificationInboxScopeTeam, TeamMemberID: "user", Limit: 100}).Validate(); err != nil {
		t.Fatalf("validate query: %v", err)
	}
	if err := (contract.NotificationInboxQuery{Scope: contract.NotificationInboxScopeDelegated, TeamMemberID: "owner", Limit: 100}).Validate(); err != nil {
		t.Fatalf("validate delegated query: %v", err)
	}
	if !errors.Is((contract.NotificationInboxQuery{Scope: contract.NotificationInboxScopeMine, TeamMemberID: "user"}).Validate(), contract.ErrInboxQueryScopeInvalid) {
		t.Fatal("expected team member outside team scope to fail")
	}
	if !errors.Is((contract.NotificationInboxQuery{Limit: 101}).Validate(), contract.ErrInboxQueryLimitInvalid) {
		t.Fatal("expected unbounded page to fail")
	}
}
