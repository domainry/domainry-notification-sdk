package remote

import (
	"context"
	"encoding/json"
	"net/http"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
)

type publisher struct{ binding *binding }
type inbox struct{ binding *binding }
type templates struct{ binding *binding }
type delivery struct{ binding *binding }
type administration struct{ binding *binding }
type systemTemplates struct{ binding *binding }
type systemSubjects struct{ binding *binding }
type systemRetention struct{ binding *binding }
type systemMigration struct{ binding *binding }

func (s systemMigration) Status(ctx context.Context) (contract.NotificationMigrationStatus, error) {
	var out contract.NotificationMigrationStatus
	err := s.binding.call(ctx, http.MethodPost, "/v1/system/migration:status", notificationsdk.UserAuthority{}, nil, &out)
	return out, err
}

func (s systemMigration) transition(ctx context.Context, path string, command contract.NotificationMigrationCommand, requireFingerprint bool) (contract.NotificationMigrationStatus, error) {
	var out contract.NotificationMigrationStatus
	if err := command.Validate(requireFingerprint); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, path, notificationsdk.UserAuthority{}, command, &out)
	return out, err
}

func (s systemMigration) Freeze(ctx context.Context, command contract.NotificationMigrationCommand) (contract.NotificationMigrationStatus, error) {
	return s.transition(ctx, "/v1/system/migration:freeze", command, false)
}

func (s systemMigration) Activate(ctx context.Context, command contract.NotificationMigrationCommand) (contract.NotificationMigrationStatus, error) {
	return s.transition(ctx, "/v1/system/migration:activate", command, true)
}

func (s systemMigration) Rollback(ctx context.Context, command contract.NotificationMigrationCommand) (contract.NotificationMigrationStatus, error) {
	return s.transition(ctx, "/v1/system/migration:rollback", command, true)
}

type systemTemplateRequest struct {
	Templates []contract.NotificationTemplate `json:"templates,omitempty"`
}

func (s systemTemplates) SyncPublished(ctx context.Context, values []contract.NotificationTemplate) error {
	return s.binding.call(ctx, http.MethodPost, "/v1/system/templates:sync-published", notificationsdk.UserAuthority{}, systemTemplateRequest{Templates: values}, nil)
}

func (s systemTemplates) ListPublished(ctx context.Context) ([]contract.NotificationTemplateRecord, error) {
	var out []contract.NotificationTemplateRecord
	err := s.binding.call(ctx, http.MethodPost, "/v1/system/templates:list-published", notificationsdk.UserAuthority{}, nil, &out)
	return out, err
}

type systemSubjectRequest struct {
	WorkspaceID string          `json:"workspace_id"`
	SubjectID   string          `json:"subject_id"`
	LegalHolds  json.RawMessage `json:"legal_holds,omitempty"`
}

func (s systemSubjects) call(ctx context.Context, path, workspaceID, subjectID string, holds json.RawMessage) (json.RawMessage, error) {
	var out json.RawMessage
	err := s.binding.call(ctx, http.MethodPost, path, notificationsdk.UserAuthority{}, systemSubjectRequest{WorkspaceID: workspaceID, SubjectID: subjectID, LegalHolds: holds}, &out)
	return out, err
}
func (s systemSubjects) PreviewSubject(ctx context.Context, workspaceID, subjectID string) (json.RawMessage, error) {
	return s.call(ctx, "/v1/system/subjects:preview", workspaceID, subjectID, nil)
}
func (s systemSubjects) ExportSubject(ctx context.Context, workspaceID, subjectID string) (json.RawMessage, error) {
	return s.call(ctx, "/v1/system/subjects:export", workspaceID, subjectID, nil)
}
func (s systemSubjects) EraseSubject(ctx context.Context, workspaceID, subjectID string, holds json.RawMessage) (json.RawMessage, error) {
	return s.call(ctx, "/v1/system/subjects:erase", workspaceID, subjectID, holds)
}

func (s systemRetention) Preview(ctx context.Context, request contract.NotificationRetentionPreviewRequest) (contract.NotificationRetentionPreview, error) {
	var out contract.NotificationRetentionPreview
	if err := request.Validate(); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/system/retention:preview", notificationsdk.UserAuthority{}, request, &out)
	return out, err
}

func (s systemRetention) ProcessBatch(ctx context.Context, request contract.NotificationRetentionBatchRequest) (contract.NotificationRetentionBatchResult, error) {
	var out contract.NotificationRetentionBatchResult
	if err := request.Validate(); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/system/retention:process-batch", notificationsdk.UserAuthority{}, request, &out)
	return out, err
}

func (s systemMigration) Export(ctx context.Context) (contract.NotificationPortableExport, error) {
	var out contract.NotificationPortableExport
	err := s.binding.call(ctx, http.MethodPost, "/v1/system/migration:export", notificationsdk.UserAuthority{}, nil, &out)
	return out, err
}

func (s systemMigration) Import(ctx context.Context, bundle contract.NotificationPortableBundle) (contract.NotificationPortableImportReceipt, error) {
	var out contract.NotificationPortableImportReceipt
	if err := bundle.ValidateEnvelope(); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/system/migration:import", notificationsdk.UserAuthority{}, bundle, &out)
	return out, err
}

type publishResponse struct {
	Event   contract.NotificationEvent `json:"event"`
	Created bool                       `json:"created"`
}

func (s publisher) PublishIntent(ctx context.Context, value contract.NotificationIntent) (contract.NotificationEvent, bool, error) {
	if err := value.Validate(); err != nil {
		return contract.NotificationEvent{}, false, err
	}
	var result publishResponse
	err := s.binding.call(ctx, http.MethodPost, "/v1/events:publish", notificationsdk.UserAuthority{}, value, &result)
	return result.Event, result.Created, err
}

type inboxQueryRequest struct {
	Query      contract.NotificationInboxQuery           `json:"query"`
	Cursor     string                                    `json:"cursor,omitempty"`
	ID         string                                    `json:"id,omitempty"`
	ActionKey  string                                    `json:"action_key,omitempty"`
	Value      bool                                      `json:"value,omitempty"`
	Surface    string                                    `json:"surface,omitempty"`
	Key        string                                    `json:"key,omitempty"`
	Delegation *contract.NotificationInboxDelegation     `json:"delegation,omitempty"`
	SavedView  *contract.NotificationInboxSavedView      `json:"saved_view,omitempty"`
	Preference *contract.NotificationRecipientPreference `json:"preference,omitempty"`
}

func validateUser(authority notificationsdk.UserAuthority) error { return authority.Validate() }
func (s inbox) List(ctx context.Context, a notificationsdk.UserAuthority, q contract.NotificationInboxQuery, cursor string) (contract.NotificationInboxPage, error) {
	var out contract.NotificationInboxPage
	if err := validateUser(a); err != nil {
		return out, err
	}
	if err := q.Validate(); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:list", a, inboxQueryRequest{Query: q, Cursor: cursor}, &out)
	return out, err
}
func (s inbox) Get(ctx context.Context, a notificationsdk.UserAuthority, id string, q contract.NotificationInboxQuery) (contract.NotificationInboxItem, error) {
	var out contract.NotificationInboxItem
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:get", a, inboxQueryRequest{ID: id, Query: q}, &out)
	return out, err
}
func (s inbox) Facets(ctx context.Context, a notificationsdk.UserAuthority, q contract.NotificationInboxQuery) (contract.NotificationInboxFacets, error) {
	var out contract.NotificationInboxFacets
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:facets", a, inboxQueryRequest{Query: q}, &out)
	return out, err
}
func (s inbox) SetRead(ctx context.Context, a notificationsdk.UserAuthority, id string, value bool) (contract.NotificationInboxItem, error) {
	var out contract.NotificationInboxItem
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:set-read", a, inboxQueryRequest{ID: id, Value: value}, &out)
	return out, err
}
func (s inbox) SetArchived(ctx context.Context, a notificationsdk.UserAuthority, id string, value bool) (contract.NotificationInboxItem, error) {
	var out contract.NotificationInboxItem
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:set-archived", a, inboxQueryRequest{ID: id, Value: value}, &out)
	return out, err
}
func (s inbox) AcknowledgeAlert(ctx context.Context, a notificationsdk.UserAuthority, id string) (contract.NotificationInboxItem, error) {
	var out contract.NotificationInboxItem
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:acknowledge", a, inboxQueryRequest{ID: id}, &out)
	return out, err
}
func (s inbox) MarkAllRead(ctx context.Context, a notificationsdk.UserAuthority, q contract.NotificationInboxQuery) (int, error) {
	var out struct {
		Count int `json:"count"`
	}
	if err := validateUser(a); err != nil {
		return 0, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:mark-all-read", a, inboxQueryRequest{Query: q}, &out)
	return out.Count, err
}
func (s inbox) ResolveAction(ctx context.Context, a notificationsdk.UserAuthority, id, key string, q contract.NotificationInboxQuery) (contract.NotificationInboxResolvedAction, error) {
	var out contract.NotificationInboxResolvedAction
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox:resolve-action", a, inboxQueryRequest{ID: id, ActionKey: key, Query: q}, &out)
	return out, err
}
func (s inbox) ListDelegations(ctx context.Context, a notificationsdk.UserAuthority, surface string) ([]contract.NotificationInboxDelegation, error) {
	var out []contract.NotificationInboxDelegation
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/delegations:list", a, inboxQueryRequest{Surface: surface}, &out)
	return out, err
}
func (s inbox) SaveDelegation(ctx context.Context, a notificationsdk.UserAuthority, v contract.NotificationInboxDelegation) (contract.NotificationInboxDelegation, error) {
	var out contract.NotificationInboxDelegation
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/delegations:save", a, inboxQueryRequest{Delegation: &v}, &out)
	return out, err
}
func (s inbox) DeleteDelegation(ctx context.Context, a notificationsdk.UserAuthority, id string) error {
	if err := validateUser(a); err != nil {
		return err
	}
	return s.binding.call(ctx, http.MethodPost, "/v1/inbox/delegations:delete", a, inboxQueryRequest{ID: id}, nil)
}
func (s inbox) ListDelegatedOwnerIDs(ctx context.Context, a notificationsdk.UserAuthority, surface string) ([]string, error) {
	var out []string
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/delegated-owners:list", a, inboxQueryRequest{Surface: surface}, &out)
	return out, err
}
func (s inbox) ListSavedViews(ctx context.Context, a notificationsdk.UserAuthority, surface string) ([]contract.NotificationInboxSavedView, error) {
	var out []contract.NotificationInboxSavedView
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/saved-views:list", a, inboxQueryRequest{Surface: surface}, &out)
	return out, err
}
func (s inbox) SaveSavedView(ctx context.Context, a notificationsdk.UserAuthority, v contract.NotificationInboxSavedView) (contract.NotificationInboxSavedView, error) {
	var out contract.NotificationInboxSavedView
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/saved-views:save", a, inboxQueryRequest{SavedView: &v}, &out)
	return out, err
}
func (s inbox) DeleteSavedView(ctx context.Context, a notificationsdk.UserAuthority, key string) error {
	if err := validateUser(a); err != nil {
		return err
	}
	return s.binding.call(ctx, http.MethodPost, "/v1/inbox/saved-views:delete", a, inboxQueryRequest{Key: key}, nil)
}
func (s inbox) GetPreference(ctx context.Context, a notificationsdk.UserAuthority, surface string) (contract.NotificationRecipientPreference, error) {
	var out contract.NotificationRecipientPreference
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/preference:get", a, inboxQueryRequest{Surface: surface}, &out)
	return out, err
}
func (s inbox) SavePreference(ctx context.Context, a notificationsdk.UserAuthority, surface string, v contract.NotificationRecipientPreference) (contract.NotificationRecipientPreference, error) {
	var out contract.NotificationRecipientPreference
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/inbox/preference:save", a, inboxQueryRequest{Surface: surface, Preference: &v}, &out)
	return out, err
}

type templateRequest struct {
	Key        string                         `json:"key,omitempty"`
	Version    int                            `json:"version,omitempty"`
	Expected   string                         `json:"expected,omitempty"`
	Locale     string                         `json:"locale,omitempty"`
	Recipients []string                       `json:"recipients,omitempty"`
	Variables  map[string]any                 `json:"variables,omitempty"`
	Template   *contract.NotificationTemplate `json:"template,omitempty"`
	Scheduled  string                         `json:"scheduled,omitempty"`
	ID         string                         `json:"id,omitempty"`
	Reason     string                         `json:"reason,omitempty"`
}

func (s templates) Capabilities(ctx context.Context, a notificationsdk.UserAuthority) ([]contract.NotificationTemplateCapability, error) {
	var out []contract.NotificationTemplateCapability
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/templates/capabilities:list", a, nil, &out)
	return out, err
}

func (s templates) List(ctx context.Context, a notificationsdk.UserAuthority) ([]contract.NotificationTemplateRecord, error) {
	var out []contract.NotificationTemplateRecord
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/templates:list", a, nil, &out)
	return out, err
}
func (s templates) Get(ctx context.Context, a notificationsdk.UserAuthority, key string) (contract.NotificationTemplateRecord, bool, error) {
	var out struct {
		Record contract.NotificationTemplateRecord `json:"record"`
		Found  bool                                `json:"found"`
	}
	if err := validateUser(a); err != nil {
		return out.Record, false, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/templates:get", a, templateRequest{Key: key}, &out)
	return out.Record, out.Found, err
}
func (s templates) ListVersions(ctx context.Context, a notificationsdk.UserAuthority, key string) ([]contract.NotificationTemplateVersion, error) {
	var out []contract.NotificationTemplateVersion
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/templates/versions:list", a, templateRequest{Key: key}, &out)
	return out, err
}
func (s templates) SaveDraft(ctx context.Context, a notificationsdk.UserAuthority, key string, v contract.NotificationTemplate, expected string) (contract.NotificationTemplateRecord, error) {
	return s.record(ctx, a, "/v1/templates:save-draft", templateRequest{Key: key, Template: &v, Expected: expected})
}
func (s templates) RestoreVersionDraft(ctx context.Context, a notificationsdk.UserAuthority, key string, version int, expected string) (contract.NotificationTemplateRecord, error) {
	return s.record(ctx, a, "/v1/templates:restore-version", templateRequest{Key: key, Version: version, Expected: expected})
}
func (s templates) Disable(ctx context.Context, a notificationsdk.UserAuthority, key, expected string) (contract.NotificationTemplateRecord, error) {
	return s.record(ctx, a, "/v1/templates:disable", templateRequest{Key: key, Expected: expected})
}
func (s templates) record(ctx context.Context, a notificationsdk.UserAuthority, path string, in templateRequest) (contract.NotificationTemplateRecord, error) {
	var out contract.NotificationTemplateRecord
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, path, a, in, &out)
	return out, err
}
func (s templates) Preview(ctx context.Context, a notificationsdk.UserAuthority, key, locale string, recipients []string, variables map[string]any) (contract.RenderedNotification, error) {
	return s.preview(ctx, a, "/v1/templates:preview", templateRequest{Key: key, Locale: locale, Recipients: recipients, Variables: variables})
}
func (s templates) PreviewTemplate(ctx context.Context, a notificationsdk.UserAuthority, v contract.NotificationTemplate, locale string, recipients []string, variables map[string]any) (contract.RenderedNotification, error) {
	return s.preview(ctx, a, "/v1/templates:preview-draft", templateRequest{Template: &v, Locale: locale, Recipients: recipients, Variables: variables})
}
func (s templates) preview(ctx context.Context, a notificationsdk.UserAuthority, path string, in templateRequest) (contract.RenderedNotification, error) {
	var out contract.RenderedNotification
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, path, a, in, &out)
	return out, err
}
func (s templates) ListPublicationRequests(ctx context.Context, a notificationsdk.UserAuthority, key string) ([]contract.NotificationPublicationRequest, error) {
	var out []contract.NotificationPublicationRequest
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/publications:list", a, templateRequest{Key: key}, &out)
	return out, err
}
func (s templates) RequestPublication(ctx context.Context, a notificationsdk.UserAuthority, key, scheduled, expected string) (contract.NotificationPublicationRequest, error) {
	return s.publication(ctx, a, "/v1/publications:request", templateRequest{Key: key, Scheduled: scheduled, Expected: expected})
}
func (s templates) ApprovePublication(ctx context.Context, a notificationsdk.UserAuthority, id string) (contract.NotificationPublicationRequest, error) {
	return s.publication(ctx, a, "/v1/publications:approve", templateRequest{ID: id})
}
func (s templates) RejectPublication(ctx context.Context, a notificationsdk.UserAuthority, id, reason string) (contract.NotificationPublicationRequest, error) {
	return s.publication(ctx, a, "/v1/publications:reject", templateRequest{ID: id, Reason: reason})
}
func (s templates) CancelPublication(ctx context.Context, a notificationsdk.UserAuthority, id string) (contract.NotificationPublicationRequest, error) {
	return s.publication(ctx, a, "/v1/publications:cancel", templateRequest{ID: id})
}
func (s templates) publication(ctx context.Context, a notificationsdk.UserAuthority, path string, in templateRequest) (contract.NotificationPublicationRequest, error) {
	var out contract.NotificationPublicationRequest
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, path, a, in, &out)
	return out, err
}

type deliveryRequest struct {
	Since  string                               `json:"since,omitempty"`
	Policy *contract.NotificationDeliveryPolicy `json:"policy,omitempty"`
}

func (s delivery) GetPolicy(ctx context.Context, a notificationsdk.UserAuthority) (contract.NotificationDeliveryPolicy, error) {
	var out contract.NotificationDeliveryPolicy
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/delivery-policy:get", a, nil, &out)
	return out, err
}
func (s delivery) SavePolicy(ctx context.Context, a notificationsdk.UserAuthority, v contract.NotificationDeliveryPolicy) (contract.NotificationDeliveryPolicy, error) {
	var out contract.NotificationDeliveryPolicy
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/delivery-policy:save", a, deliveryRequest{Policy: &v}, &out)
	return out, err
}
func (s delivery) ListRecipientPreferences(ctx context.Context, a notificationsdk.UserAuthority) ([]contract.NotificationRecipientPreference, error) {
	var out []contract.NotificationRecipientPreference
	if err := validateUser(a); err != nil {
		return nil, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/recipient-preferences:list", a, nil, &out)
	return out, err
}
func (s delivery) SaveRecipientPreference(ctx context.Context, a notificationsdk.UserAuthority, v contract.NotificationRecipientPreference) (contract.NotificationRecipientPreference, error) {
	var out contract.NotificationRecipientPreference
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/recipient-preferences:save", a, inboxQueryRequest{Preference: &v}, &out)
	return out, err
}
func (s delivery) Metrics(ctx context.Context, a notificationsdk.UserAuthority, since string) (contract.NotificationDeliveryMetrics, error) {
	var out contract.NotificationDeliveryMetrics
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/delivery-metrics:get", a, deliveryRequest{Since: since}, &out)
	return out, err
}

type administrationRequest struct {
	Since string `json:"since,omitempty"`
}

func (s administration) GovernanceCatalog(ctx context.Context, a notificationsdk.UserAuthority) (contract.NotificationGovernanceCatalog, error) {
	var out contract.NotificationGovernanceCatalog
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/governance/catalog:get", a, nil, &out)
	return out, err
}
func (s administration) InboxGovernanceMetrics(ctx context.Context, a notificationsdk.UserAuthority, since string) (contract.NotificationInboxGovernanceMetrics, error) {
	var out contract.NotificationInboxGovernanceMetrics
	if err := validateUser(a); err != nil {
		return out, err
	}
	err := s.binding.call(ctx, http.MethodPost, "/v1/governance/inbox-metrics:get", a, administrationRequest{Since: since}, &out)
	return out, err
}

var _ notificationsdk.Publisher = publisher{}
var _ notificationsdk.Inbox = inbox{}
var _ notificationsdk.Templates = templates{}
var _ notificationsdk.Delivery = delivery{}
var _ notificationsdk.Administration = administration{}
