// Package deliverygateway defines the authenticated Integration acceptance
// boundary used by standalone Notification SaaS. It contains no provider
// credentials and does not expose Runtime persistence.
package deliverygateway

import (
	"context"
	"fmt"
	"strings"

	notificationsdk "github.com/domainry/domainry-notification-sdk"
	"github.com/domainry/domainry-notification-sdk/contract"
)

const AcceptPath = "/v1/notification-deliveries:accept"

type Fallback struct {
	ConnectorKey  string                        `json:"connector_key"`
	ConnectionKey string                        `json:"connection_key,omitempty"`
	Operation     string                        `json:"operation"`
	Rendered      contract.RenderedNotification `json:"rendered"`
}

type Request struct {
	RequestID     string                        `json:"request_id"`
	WorkspaceID   string                        `json:"workspace_id"`
	PlanID        string                        `json:"plan_id"`
	EventID       string                        `json:"event_id"`
	Channel       string                        `json:"channel"`
	ConnectorKey  string                        `json:"connector_key"`
	ConnectionKey string                        `json:"connection_key,omitempty"`
	Operation     string                        `json:"operation"`
	DedupeKey     string                        `json:"dedupe_key"`
	DeliverAfter  string                        `json:"deliver_after,omitempty"`
	CreatedAt     string                        `json:"created_at"`
	Rendered      contract.RenderedNotification `json:"rendered"`
	Fallbacks     []Fallback                    `json:"fallbacks,omitempty"`
}

func (r Request) Validate(application notificationsdk.ApplicationRef) error {
	if err := application.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.RequestID) == "" || strings.TrimSpace(r.PlanID) == "" || strings.TrimSpace(r.EventID) == "" || strings.TrimSpace(r.WorkspaceID) != application.WorkspaceID || strings.TrimSpace(r.Channel) == "" || strings.TrimSpace(r.ConnectorKey) == "" || strings.TrimSpace(r.Operation) == "" || strings.TrimSpace(r.DedupeKey) == "" || strings.TrimSpace(r.CreatedAt) == "" {
		return fmt.Errorf("notification Delivery Gateway request identity and scope are required")
	}
	return nil
}

type Receipt struct {
	RequestID string `json:"request_id"`
	MessageID string `json:"message_id"`
}

type Gateway interface {
	Dispatch(context.Context, notificationsdk.ApplicationRef, Request) (Receipt, error)
}
