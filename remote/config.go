// Package remote implements the Notification SaaS SDK Factory.
package remote

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	identitysdk "github.com/domainry/domainry-identity-sdk"
	identityremote "github.com/domainry/domainry-identity-sdk/remote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type ContextHeaderProvider func(context.Context) http.Header
type RetryPolicy struct {
	MaxAttempts                int
	InitialBackoff, MaxBackoff time.Duration
}
type CircuitBreakerPolicy struct {
	FailureThreshold int
	OpenDuration     time.Duration
}

type Config struct {
	BaseURL, ServiceCredential string
	CapabilityContractSHA256   string
	ServiceTokens              ServiceTokenSource
	HTTPClient                 *http.Client
	RequestTimeout             time.Duration
	Retry                      RetryPolicy
	CircuitBreaker             CircuitBreakerPolicy
	ContextHeaders             ContextHeaderProvider
}

func ConfigFromEnvironment() Config {
	config := Config{
		BaseURL: strings.TrimSpace(os.Getenv("NOTIFICATION_SAAS_URL")), ServiceCredential: strings.TrimSpace(os.Getenv("NOTIFICATION_SAAS_SERVICE_CREDENTIAL")),
		CapabilityContractSHA256: strings.TrimSpace(os.Getenv("NOTIFICATION_CAPABILITY_CONTRACT_SHA256")), ContextHeaders: OpenTelemetryContextHeaders,
	}
	identityConfig := identityremote.ConfigFromEnvironment()
	if identityConfig.Endpoint != "" && identityConfig.ServiceAccessToken != "" {
		config.ServiceTokens = NewIdentityServiceTokenSource(identityremote.NewFactory(identityConfig))
	}
	return config
}

type ServiceToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

type ServiceTokenSource interface {
	Token(context.Context, identitysdk.ApplicationRef, identitysdk.ApplicationServiceGrant) (ServiceToken, error)
}

// OpenTelemetryContextHeaders propagates the active W3C trace/baggage context
// without coupling generated composition to a specific exporter.
func OpenTelemetryContextHeaders(ctx context.Context) http.Header {
	header := http.Header{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
	return header
}

func normalizedConfig(c Config) Config {
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = 10 * time.Second
	}
	if c.Retry.MaxAttempts <= 0 {
		c.Retry.MaxAttempts = 2
	}
	if c.Retry.InitialBackoff <= 0 {
		c.Retry.InitialBackoff = 50 * time.Millisecond
	}
	if c.Retry.MaxBackoff <= 0 {
		c.Retry.MaxBackoff = 500 * time.Millisecond
	}
	if c.Retry.MaxBackoff < c.Retry.InitialBackoff {
		c.Retry.MaxBackoff = c.Retry.InitialBackoff
	}
	if c.CircuitBreaker.FailureThreshold <= 0 {
		c.CircuitBreaker.FailureThreshold = 5
	}
	if c.CircuitBreaker.OpenDuration <= 0 {
		c.CircuitBreaker.OpenDuration = 5 * time.Second
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}
	return c
}
