package contract

import (
	"errors"
	"testing"
)

func TestValidateEventTypesContract(t *testing.T) {
	valid := NotificationEventType{
		Key: "workflow.task.assigned", Source: "workflow", Category: "approval",
		DefaultSeverity: "info", Surfaces: []string{"business_workspace"},
		TemplateKey: "workflow.task.assigned.in_app", DefaultLocale: "en-US",
		Locales:   map[string]NotificationInboxEventTypeContent{"en-US": {Title: "Assigned", Body: "Open {{task_title}}", ActionLabels: map[string]string{"workflow.task.open": "Open"}}},
		Variables: []NotificationTemplateVariable{{Key: "task_title", Type: "string", Required: true}},
		Actions:   []NotificationInboxActionDescriptor{{Key: "workflow.task.open", Kind: "route", ResourceType: "workflow_task", SurfaceRoutes: map[string]string{"business_workspace": "workflow.task.detail"}}},
		Version:   1, Status: "published",
	}
	if err := ValidateEventTypes([]NotificationEventType{valid}, []NotificationRule{{EventTypeKey: valid.Key, Channels: []NotificationRuleChannel{{Channel: "in_app"}}}}); err != nil {
		t.Fatalf("valid catalog: %v", err)
	}
	if err := ValidateEventTypes([]NotificationEventType{valid, valid}, nil); catalogErrorCode(err) != "backend.notification.event_type_duplicate" {
		t.Fatalf("duplicate code=%q err=%v", catalogErrorCode(err), err)
	}
	unsafe := valid
	unsafe.Variables = []NotificationTemplateVariable{{Key: "access_token", Type: "string"}}
	unsafe.Locales = map[string]NotificationInboxEventTypeContent{"en-US": {Title: "Assigned", Body: "Open"}}
	unsafe.Actions = nil
	if _, err := ValidateEventType(unsafe); catalogErrorCode(err) != "backend.notification.event_type_variable_unsafe" {
		t.Fatalf("unsafe code=%q err=%v", catalogErrorCode(err), err)
	}
	badRule := NotificationRule{EventTypeKey: valid.Key, Channels: []NotificationRuleChannel{{Channel: "email", TemplateKey: "mail", ConnectorKey: "connector", Operation: "send", DeliveryMode: "digest"}}}
	if err := ValidateEventTypes([]NotificationEventType{valid}, []NotificationRule{badRule}); catalogErrorCode(err) != "backend.notification.rule_channel_digest_invalid" {
		t.Fatalf("digest code=%q err=%v", catalogErrorCode(err), err)
	}
}

func TestSupportedProductSurfacesAreImmutable(t *testing.T) {
	first := SupportedProductSurfaces()
	if len(first) != 2 || first[0] != "business_workspace" || first[1] != "consumer_portal" {
		t.Fatalf("surfaces=%v", first)
	}
	first[0] = "mutated"
	if next := SupportedProductSurfaces(); next[0] != "business_workspace" {
		t.Fatalf("supported surface catalog was mutated: %v", next)
	}
}

func catalogErrorCode(err error) string {
	var target *CatalogValidationError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}
