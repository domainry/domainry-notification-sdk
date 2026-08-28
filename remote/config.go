// Package remote implements the Notification SaaS SDK Factory.
package remote

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
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
	HTTPClient                 *http.Client
	RequestTimeout             time.Duration
	Retry                      RetryPolicy
	CircuitBreaker             CircuitBreakerPolicy
	ContextHeaders             ContextHeaderProvider
}

func ConfigFromEnvironment() Config {
	return Config{BaseURL: strings.TrimSpace(os.Getenv("NOTIFICATION_SAAS_URL")), ServiceCredential: strings.TrimSpace(os.Getenv("NOTIFICATION_SAAS_SERVICE_CREDENTIAL"))}
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
