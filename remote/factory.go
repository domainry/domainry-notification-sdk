package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/domainry/domainry-foundation/modulecapability"
	identitysdk "github.com/domainry/domainry-identity-sdk"
	notificationsdk "github.com/domainry/domainry-notification-sdk"
)

type Factory struct{ config Config }

func NewFactory(config Config) *Factory { return &Factory{config: config} }

func (f *Factory) Open(ctx context.Context, application notificationsdk.ApplicationRef) (notificationsdk.Binding, error) {
	if err := application.Validate(); err != nil {
		return nil, err
	}
	config := normalizedConfig(f.config)
	if err := modulecapability.ValidateRemoteExpectation("notification", config.CapabilityContractSHA256); err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" || strings.TrimSpace(config.ServiceCredential) == "" && config.ServiceTokens == nil {
		return nil, remoteError(http.StatusInternalServerError, "notification.remote_config_incomplete", false, nil)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, remoteError(http.StatusInternalServerError, "notification.remote_url_invalid", false, err)
	}
	b := &binding{baseURL: baseURL, serviceCredential: config.ServiceCredential, serviceTokens: config.ServiceTokens, cachedTokens: map[string]ServiceToken{}, application: application, client: config.HTTPClient, requestTimeout: config.RequestTimeout, retry: config.Retry, contextHeaders: config.ContextHeaders, breaker: &circuitBreaker{policy: config.CircuitBreaker}}
	var descriptor notificationsdk.Descriptor
	if err := b.call(ctx, http.MethodGet, "/v1/descriptor", notificationsdk.UserAuthority{}, nil, &descriptor); err != nil {
		return nil, fmt.Errorf("discover Notification SaaS: %w", err)
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	if descriptor.Mode != notificationsdk.DeploymentModeSaaS {
		return nil, remoteError(http.StatusBadGateway, "notification.remote_mode_mismatch", false, nil)
	}
	if descriptor.Audience != "" && descriptor.Audience != application.ApplicationKey {
		return nil, remoteError(http.StatusForbidden, "notification.application_scope_mismatch", false, nil)
	}
	b.descriptor = descriptor
	capability, err := openCapabilityBinding(ctx, b, config.CapabilityContractSHA256)
	if err != nil {
		return nil, fmt.Errorf("discover Notification capability contract: %w", err)
	}
	b.capability = capability
	return b, nil
}

type binding struct {
	baseURL, serviceCredential string
	serviceTokens              ServiceTokenSource
	tokenMu                    sync.Mutex
	cachedTokens               map[string]ServiceToken
	application                notificationsdk.ApplicationRef
	descriptor                 notificationsdk.Descriptor
	capability                 *remoteCapabilityBinding
	client                     *http.Client
	requestTimeout             time.Duration
	retry                      RetryPolicy
	contextHeaders             ContextHeaderProvider
	breaker                    *circuitBreaker
}

func (b *binding) Descriptor() notificationsdk.Descriptor { return b.descriptor }
func (b *binding) CapabilitySummary(ctx context.Context) (modulecapability.ModuleSummary, error) {
	return b.capability.CapabilitySummary(ctx)
}
func (b *binding) CapabilityCategory(ctx context.Context, key string) (modulecapability.CategoryDocument, error) {
	return b.capability.CapabilityCategory(ctx, key)
}
func (b *binding) ValidateCapabilityCandidate(ctx context.Context, request modulecapability.ValidationRequest) (modulecapability.ValidationResult, error) {
	return b.capability.ValidateCapabilityCandidate(ctx, request)
}
func (b *binding) Publisher() notificationsdk.Publisher           { return publisher{binding: b} }
func (b *binding) Inbox() notificationsdk.Inbox                   { return inbox{binding: b} }
func (b *binding) Templates() notificationsdk.Templates           { return templates{binding: b} }
func (b *binding) Delivery() notificationsdk.Delivery             { return delivery{binding: b} }
func (b *binding) Administration() notificationsdk.Administration { return administration{binding: b} }
func (b *binding) SystemTemplates() notificationsdk.SystemTemplates {
	return systemTemplates{binding: b}
}
func (b *binding) SystemSubjects() notificationsdk.SystemSubjects { return systemSubjects{binding: b} }
func (b *binding) SystemRetention() notificationsdk.SystemRetention {
	return systemRetention{binding: b}
}
func (b *binding) SystemMigration() notificationsdk.SystemMigration {
	return systemMigration{binding: b}
}
func (b *binding) LocalWorkers() (notificationsdk.LocalWorkers, bool) { return nil, false }
func (b *binding) Close(context.Context) error                        { return nil }

func (b *binding) call(ctx context.Context, method, path string, authority notificationsdk.UserAuthority, input, output any) error {
	var encoded []byte
	if input != nil {
		var err error
		encoded, err = json.Marshal(input)
		if err != nil {
			return remoteError(http.StatusInternalServerError, "notification.request_encode_failed", false, err)
		}
	}
	if !b.breaker.allow(time.Now()) {
		return remoteError(http.StatusServiceUnavailable, "notification.remote_circuit_open", true, nil)
	}
	attempts := 1
	if retryableRequest(method, path) {
		attempts = b.retry.MaxAttempts
	}
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		status, err := b.perform(ctx, method, path, authority, encoded, output)
		failed := err != nil && (status == 0 || retryableStatus(status))
		if err == nil {
			b.breaker.observe(time.Now(), false)
			return nil
		}
		last = err
		if !failed || attempt == attempts {
			b.breaker.observe(time.Now(), failed)
			return err
		}
		if err := waitRetry(ctx, retryDelay(b.retry, attempt)); err != nil {
			b.breaker.observe(time.Now(), false)
			return remoteError(http.StatusGatewayTimeout, "notification.remote_retry_cancelled", false, err)
		}
	}
	return last
}

func (b *binding) perform(parent context.Context, method, path string, authority notificationsdk.UserAuthority, encoded []byte, output any) (int, error) {
	ctx, cancel := context.WithTimeout(parent, b.requestTimeout)
	defer cancel()
	var body io.Reader
	if encoded != nil {
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, b.baseURL+path, body)
	if err != nil {
		return 0, remoteError(http.StatusInternalServerError, "notification.request_build_failed", false, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	grant, err := notificationsdk.ServiceGrantForRequest(method, path)
	if err != nil {
		return 0, err
	}
	serviceToken, err := b.serviceToken(parent, grant)
	if err != nil {
		return 0, err
	}
	request.Header.Set("X-Domainry-Service-Credential", serviceToken)
	request.Header.Set("X-Domainry-Tenant-ID", b.application.TenantID)
	request.Header.Set("X-Domainry-Workspace-ID", b.application.WorkspaceID)
	request.Header.Set("X-Domainry-Application-Key", b.application.ApplicationKey)
	if strings.TrimSpace(authority.AccessToken) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(authority.AccessToken))
	}
	if strings.TrimSpace(authority.Surface) != "" {
		request.Header.Set("X-Domainry-Product-Surface", strings.TrimSpace(authority.Surface))
	}
	if b.contextHeaders != nil {
		copyContextHeaders(request.Header, b.contextHeaders(parent))
	}
	response, err := b.client.Do(request)
	if err != nil {
		return 0, remoteError(http.StatusServiceUnavailable, "notification.remote_unavailable", true, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var envelope notificationsdk.Error
		if decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&envelope); decodeErr != nil || envelope.Code == "" {
			return response.StatusCode, remoteError(response.StatusCode, "notification.remote_rejected", retryableStatus(response.StatusCode), decodeErr)
		}
		envelope.StatusCode = response.StatusCode
		return response.StatusCode, &envelope
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return response.StatusCode, nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(output); err != nil {
		return response.StatusCode, remoteError(http.StatusBadGateway, "notification.response_decode_failed", false, err)
	}
	return response.StatusCode, nil
}

func (b *binding) serviceToken(ctx context.Context, grant identitysdk.ApplicationServiceGrant) (string, error) {
	if b.serviceTokens == nil {
		return strings.TrimSpace(b.serviceCredential), nil
	}
	b.tokenMu.Lock()
	defer b.tokenMu.Unlock()
	key := string(grant.Resource) + "\x00" + string(grant.Action)
	cached := b.cachedTokens[key]
	if strings.TrimSpace(cached.AccessToken) != "" && cached.ExpiresAt.After(time.Now().UTC().Add(30*time.Second)) {
		return cached.AccessToken, nil
	}
	token, err := b.serviceTokens.Token(ctx, identitysdk.ApplicationRef{TenantID: identitysdk.TenantID(b.application.TenantID), WorkspaceID: identitysdk.WorkspaceID(b.application.WorkspaceID), ApplicationKey: identitysdk.ApplicationKey(b.application.ApplicationKey)}, grant)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token.AccessToken) == "" || !token.ExpiresAt.After(time.Now().UTC()) {
		return "", remoteError(http.StatusBadGateway, "notification.identity_service_token_invalid", false, nil)
	}
	b.cachedTokens[key] = token
	return token.AccessToken, nil
}

func remoteError(status int, code string, retryable bool, cause error) error {
	return &notificationsdk.Error{StatusCode: status, Code: code, Retryable: retryable, Cause: cause}
}

var _ notificationsdk.Factory = (*Factory)(nil)
var _ notificationsdk.Binding = (*binding)(nil)
