package contract

import (
	"errors"
	"strings"
)

var (
	ErrIntentScopeRequired      = errors.New("notification intent workspace is required")
	ErrIntentIdentityRequired   = errors.New("notification intent identity is incomplete")
	ErrIntentSubjectRequired    = errors.New("notification intent subject is incomplete")
	ErrIntentAudienceRequired   = errors.New("notification intent audience is empty")
	ErrIntentAudienceTooLarge   = errors.New("notification intent audience exceeds the synchronous limit")
	ErrInboxQueryLimitInvalid   = errors.New("notification inbox query limit is invalid")
	ErrInboxQueryScopeInvalid   = errors.New("notification inbox query scope is invalid")
	ErrInboxQueryMailboxInvalid = errors.New("notification inbox query mailbox is invalid")
)

const MaximumExplicitRecipients = 500

func (value NotificationIntent) Validate() error {
	if strings.TrimSpace(value.WorkspaceID) == "" {
		return ErrIntentScopeRequired
	}
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.SourceEventID) == "" || strings.TrimSpace(value.EventType) == "" || strings.TrimSpace(value.OccurredAt) == "" {
		return ErrIntentIdentityRequired
	}
	if (strings.TrimSpace(value.SubjectType) == "") != (strings.TrimSpace(value.SubjectID) == "") || (strings.TrimSpace(value.SubjectID) != "" && strings.TrimSpace(value.SubjectVersion) == "") {
		return ErrIntentSubjectRequired
	}
	if len(value.RecipientUserIDs) == 0 && len(value.AudienceResolverKeys) == 0 {
		return ErrIntentAudienceRequired
	}
	if len(value.RecipientUserIDs) > MaximumExplicitRecipients {
		return ErrIntentAudienceTooLarge
	}
	seen := make(map[string]struct{}, len(value.RecipientUserIDs))
	for _, recipient := range value.RecipientUserIDs {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			return ErrIntentAudienceRequired
		}
		if _, exists := seen[recipient]; exists {
			return ErrIntentAudienceRequired
		}
		seen[recipient] = struct{}{}
	}
	return nil
}

func (value NotificationInboxQuery) Validate() error {
	if value.Limit < 0 || value.Limit > 100 {
		return ErrInboxQueryLimitInvalid
	}
	switch strings.TrimSpace(value.Scope) {
	case "", NotificationInboxScopeMine, NotificationInboxScopeTeam, NotificationInboxScopeDelegated:
	default:
		return ErrInboxQueryScopeInvalid
	}
	if value.TeamMemberID != "" && value.Scope != NotificationInboxScopeTeam && value.Scope != NotificationInboxScopeDelegated {
		return ErrInboxQueryScopeInvalid
	}
	switch strings.TrimSpace(value.Mailbox) {
	case "", NotificationMailboxInbox, NotificationMailboxUnread, NotificationMailboxActionRequired, NotificationMailboxArchived:
	default:
		return ErrInboxQueryMailboxInvalid
	}
	return nil
}
