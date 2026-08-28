package notificationsdk

import (
	"net/http"

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
	switch path {
	case "/v1/descriptor":
		grant = serviceGrant("notification_service", "discover")
	case "/v1/events:publish":
		grant = serviceGrant("notification_event", "publish")
	case "/v1/inbox:list", "/v1/inbox:get", "/v1/inbox:facets", "/v1/inbox/delegations:list", "/v1/inbox/delegated-owners:list", "/v1/inbox/saved-views:list", "/v1/inbox/preference:get":
		grant = serviceGrant("notification_inbox", "read")
	case "/v1/inbox:resolve-action":
		grant = serviceGrant("notification_inbox", "act")
	case "/v1/inbox:set-read", "/v1/inbox:set-archived", "/v1/inbox:acknowledge", "/v1/inbox:mark-all-read", "/v1/inbox/delegations:save", "/v1/inbox/delegations:delete", "/v1/inbox/saved-views:save", "/v1/inbox/saved-views:delete", "/v1/inbox/preference:save":
		grant = serviceGrant("notification_inbox", "update")
	case "/v1/templates/capabilities:list", "/v1/templates:list", "/v1/templates:get", "/v1/templates/versions:list", "/v1/templates:preview", "/v1/system/templates:list-published":
		grant = serviceGrant("notification_template", "read")
	case "/v1/templates:save-draft", "/v1/templates:preview-draft", "/v1/templates:restore-version", "/v1/system/templates:sync-published":
		grant = serviceGrant("notification_template", "draft")
	case "/v1/templates:disable":
		grant = serviceGrant("notification_template", "disable")
	case "/v1/publications:list":
		grant = serviceGrant("notification_publication", "read")
	case "/v1/publications:request":
		grant = serviceGrant("notification_publication", "request")
	case "/v1/publications:approve":
		grant = serviceGrant("notification_publication", "approve")
	case "/v1/publications:reject":
		grant = serviceGrant("notification_publication", "reject")
	case "/v1/publications:cancel":
		grant = serviceGrant("notification_publication", "cancel")
	case "/v1/delivery-policy:get":
		grant = serviceGrant("notification_delivery_policy", "read")
	case "/v1/delivery-policy:save":
		grant = serviceGrant("notification_delivery_policy", "update")
	case "/v1/recipient-preferences:list":
		grant = serviceGrant("notification_preference", "read")
	case "/v1/recipient-preferences:save":
		grant = serviceGrant("notification_preference", "update")
	case "/v1/delivery-metrics:get", "/v1/governance/catalog:get", "/v1/governance/inbox-metrics:get":
		grant = serviceGrant("notification_governance", "read")
	case "/v1/system/subjects:preview", "/v1/system/subjects:export":
		grant = serviceGrant("notification_governance", "export")
	case "/v1/system/subjects:erase":
		grant = serviceGrant("notification_governance", "erase")
	case "/v1/system/retention:preview", "/v1/system/retention:process-batch":
		grant = serviceGrant("notification_governance", "retention")
	case "/v1/system/migration:status", "/v1/system/migration:freeze", "/v1/system/migration:export", "/v1/system/migration:import", "/v1/system/migration:activate", "/v1/system/migration:rollback":
		grant = serviceGrant("notification_governance", "migrate")
	default:
		return identitysdk.ApplicationServiceGrant{}, &Error{StatusCode: http.StatusNotFound, Code: "notification.route_not_found"}
	}
	return grant, nil
}

func serviceGrant(resource, action string) identitysdk.ApplicationServiceGrant {
	return identitysdk.ApplicationServiceGrant{Resource: identitysdk.ResourceType(resource), Action: identitysdk.Action(action)}
}
