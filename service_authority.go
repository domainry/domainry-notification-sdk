package notificationsdk

import (
	"net/http"
	"strings"

	"github.com/domainry/domainry-foundation/modulecapability"
	identitysdk "github.com/domainry/domainry-identity-sdk"
)

// ServiceGrantForRequest is the single protocol mapping used by both the
// Remote client and SaaS server. A short-lived Identity service token can
// authorize only the exact Notification capability required by one route.
func ServiceGrantForRequest(method, path string) (identitysdk.ApplicationServiceGrant, error) {
	if method != http.MethodGet && method != http.MethodPost {
		return identitysdk.ApplicationServiceGrant{}, &Error{StatusCode: http.StatusMethodNotAllowed, Code: "notification.method_not_allowed"}
	}
	grant := identitysdk.ApplicationServiceGrant{}
	if (method == http.MethodGet && (path == modulecapability.SummaryPath || strings.HasPrefix(path, modulecapability.CategoriesPath))) ||
		(method == http.MethodPost && path == modulecapability.ValidationPath) {
		return serviceGrant("notification_service", "discover"), nil
	}
	switch path {
	case "/notification/v1/descriptor":
		grant = serviceGrant("notification_service", "discover")
	case "/notification/v1/events:publish":
		grant = serviceGrant("notification_event", "publish")
	case "/notification/v1/inbox:list", "/notification/v1/inbox:get", "/notification/v1/inbox:facets", "/notification/v1/inbox/delegations:list", "/notification/v1/inbox/delegated-owners:list", "/notification/v1/inbox/saved-views:list", "/notification/v1/inbox/preference:get":
		grant = serviceGrant("notification_inbox", "read")
	case "/notification/v1/inbox:resolve-action":
		grant = serviceGrant("notification_inbox", "act")
	case "/notification/v1/inbox:set-read", "/notification/v1/inbox:set-archived", "/notification/v1/inbox:acknowledge", "/notification/v1/inbox:mark-all-read", "/notification/v1/inbox/delegations:save", "/notification/v1/inbox/delegations:delete", "/notification/v1/inbox/saved-views:save", "/notification/v1/inbox/saved-views:delete", "/notification/v1/inbox/preference:save":
		grant = serviceGrant("notification_inbox", "update")
	case "/notification/v1/templates/capabilities:list", "/notification/v1/templates:list", "/notification/v1/templates:get", "/notification/v1/templates/versions:list", "/notification/v1/templates:preview", "/notification/v1/system/templates:list-published":
		grant = serviceGrant("notification_template", "read")
	case "/notification/v1/templates:save-draft", "/notification/v1/templates:preview-draft", "/notification/v1/templates:restore-version", "/notification/v1/system/templates:sync-published":
		grant = serviceGrant("notification_template", "draft")
	case "/notification/v1/templates:disable":
		grant = serviceGrant("notification_template", "disable")
	case "/notification/v1/publications:list":
		grant = serviceGrant("notification_publication", "read")
	case "/notification/v1/publications:request":
		grant = serviceGrant("notification_publication", "request")
	case "/notification/v1/publications:approve":
		grant = serviceGrant("notification_publication", "approve")
	case "/notification/v1/publications:reject":
		grant = serviceGrant("notification_publication", "reject")
	case "/notification/v1/publications:cancel":
		grant = serviceGrant("notification_publication", "cancel")
	case "/notification/v1/delivery-policy:get":
		grant = serviceGrant("notification_delivery_policy", "read")
	case "/notification/v1/delivery-policy:save":
		grant = serviceGrant("notification_delivery_policy", "update")
	case "/notification/v1/recipient-preferences:list":
		grant = serviceGrant("notification_preference", "read")
	case "/notification/v1/recipient-preferences:save":
		grant = serviceGrant("notification_preference", "update")
	case "/notification/v1/delivery-metrics:get", "/notification/v1/governance/catalog:get", "/notification/v1/governance/inbox-metrics:get":
		grant = serviceGrant("notification_governance", "read")
	case "/notification/v1/system/subjects:preview", "/notification/v1/system/subjects:export":
		grant = serviceGrant("notification_governance", "export")
	case "/notification/v1/system/subjects:erase":
		grant = serviceGrant("notification_governance", "erase")
	case "/notification/v1/system/retention:preview", "/notification/v1/system/retention:process-batch":
		grant = serviceGrant("notification_governance", "retention")
	case "/notification/v1/system/migration:status", "/notification/v1/system/migration:freeze", "/notification/v1/system/migration:export", "/notification/v1/system/migration:import", "/notification/v1/system/migration:activate", "/notification/v1/system/migration:rollback":
		grant = serviceGrant("notification_governance", "migrate")
	default:
		return identitysdk.ApplicationServiceGrant{}, &Error{StatusCode: http.StatusNotFound, Code: "notification.route_not_found"}
	}
	return grant, nil
}

func serviceGrant(resource, action string) identitysdk.ApplicationServiceGrant {
	return identitysdk.ApplicationServiceGrant{Resource: identitysdk.ResourceType(resource), Action: identitysdk.Action(action)}
}
