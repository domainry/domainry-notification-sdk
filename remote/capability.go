package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/domainry/domainry-foundation/modulecapability"
	notificationsdk "github.com/domainry/domainry-notification-sdk"
)

// remoteCapabilityBinding deliberately uses Notification's existing call
// path so discovery and owner validation inherit its service-token, timeout,
// retry, tracing and circuit-breaker behavior. Disclosure is fetched and
// verified once when the application Binding opens.
type remoteCapabilityBinding struct {
	binding    *binding
	summary    modulecapability.ModuleSummary
	categories map[string]modulecapability.CategoryDocument
}

func openCapabilityBinding(ctx context.Context, binding *binding, expectedSHA256 string) (*remoteCapabilityBinding, error) {
	if binding == nil {
		return nil, &notificationsdk.Error{StatusCode: http.StatusServiceUnavailable, Code: "notification.binding_unavailable"}
	}
	if err := modulecapability.ValidateRemoteExpectation("notification", expectedSHA256); err != nil {
		return nil, err
	}
	value := &remoteCapabilityBinding{binding: binding}
	if err := binding.call(ctx, http.MethodGet, modulecapability.SummaryPath, notificationsdk.UserAuthority{}, nil, &value.summary); err != nil {
		return nil, err
	}
	if err := modulecapability.ValidateModuleSummary(value.summary); err != nil || value.summary.Identity.Key != "notification" {
		return nil, &notificationsdk.Error{StatusCode: http.StatusConflict, Code: "notification.capability_contract_mismatch", Cause: err}
	}
	if value.summary.Identity.ContractSHA256 != strings.TrimSpace(expectedSHA256) {
		return nil, &notificationsdk.Error{StatusCode: http.StatusConflict, Code: "notification.capability_contract_mismatch"}
	}
	value.categories = make(map[string]modulecapability.CategoryDocument, len(value.summary.Categories))
	documents := make([]modulecapability.CategoryDocument, 0, len(value.summary.Categories))
	for _, category := range value.summary.Categories {
		var document modulecapability.CategoryDocument
		if err := binding.call(ctx, http.MethodGet, modulecapability.CategoriesPath+category.Key, notificationsdk.UserAuthority{}, nil, &document); err != nil {
			return nil, err
		}
		value.categories[category.Key] = document
		documents = append(documents, document)
	}
	if err := modulecapability.ValidateBundle(value.summary, documents); err != nil {
		return nil, &notificationsdk.Error{StatusCode: http.StatusConflict, Code: "notification.capability_contract_mismatch", Cause: err}
	}
	return value, nil
}

func (value *remoteCapabilityBinding) CapabilitySummary(context.Context) (modulecapability.ModuleSummary, error) {
	return capabilityClone(value.summary)
}

func (value *remoteCapabilityBinding) CapabilityCategory(_ context.Context, key string) (modulecapability.CategoryDocument, error) {
	document, found := value.categories[strings.TrimSpace(key)]
	if !found {
		return modulecapability.CategoryDocument{}, &modulecapability.Error{StatusCode: http.StatusNotFound, Code: "module_capability.category_not_found"}
	}
	return capabilityClone(document)
}

func (value *remoteCapabilityBinding) ValidateCapabilityCandidate(ctx context.Context, request modulecapability.ValidationRequest) (modulecapability.ValidationResult, error) {
	if err := modulecapability.ValidateValidationRequest(request); err != nil {
		return modulecapability.ValidationResult{}, &modulecapability.Error{StatusCode: http.StatusBadRequest, Code: "module_capability.validation_request_invalid", Message: err.Error()}
	}
	var result modulecapability.ValidationResult
	if err := value.binding.call(ctx, http.MethodPost, modulecapability.ValidationPath, notificationsdk.UserAuthority{}, request, &result); err != nil {
		return modulecapability.ValidationResult{}, err
	}
	if err := modulecapability.ValidateValidationResult(result, value.summary.Identity, request.CategoryKey); err != nil {
		return modulecapability.ValidationResult{}, &notificationsdk.Error{StatusCode: http.StatusConflict, Code: "notification.capability_contract_mismatch", Cause: err}
	}
	return result, nil
}

func capabilityClone[T any](source T) (T, error) {
	var result T
	payload, err := modulecapability.CanonicalJSON(source)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return result, err
	}
	return result, nil
}

var _ modulecapability.Binding = (*remoteCapabilityBinding)(nil)
