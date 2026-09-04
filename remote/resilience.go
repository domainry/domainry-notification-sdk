package remote

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

type circuitBreaker struct {
	mu          sync.Mutex
	policy      CircuitBreakerPolicy
	failures    int
	openedUntil time.Time
	probeActive bool
}

func (b *circuitBreaker) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openedUntil.IsZero() {
		return true
	}
	if now.Before(b.openedUntil) || b.probeActive {
		return false
	}
	b.probeActive = true
	return true
}
func (b *circuitBreaker) observe(now time.Time, failed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !failed {
		b.failures = 0
		b.openedUntil = time.Time{}
		b.probeActive = false
		return
	}
	if b.probeActive || b.failures+1 >= b.policy.FailureThreshold {
		b.failures = b.policy.FailureThreshold
		b.openedUntil = now.Add(b.policy.OpenDuration)
		b.probeActive = false
		return
	}
	b.failures++
}

func retryableRequest(method, path string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions || method == http.MethodPost && path == "/notification/v1/events:publish"
}
func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}
func retryDelay(policy RetryPolicy, attempt int) time.Duration {
	delay := policy.InitialBackoff
	for n := 1; n < attempt && delay < policy.MaxBackoff; n++ {
		delay *= 2
		if delay > policy.MaxBackoff {
			delay = policy.MaxBackoff
		}
	}
	return delay
}
func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var protectedHeaders = map[string]bool{"Authorization": true, "Content-Length": true, "Content-Type": true, "Cookie": true, "Host": true, "X-Domainry-Service-Credential": true, "X-Domainry-Tenant-Id": true, "X-Domainry-Workspace-Id": true, "X-Domainry-Application-Key": true}

func copyContextHeaders(target, source http.Header) {
	for key, values := range source {
		canonical := http.CanonicalHeaderKey(strings.TrimSpace(key))
		if canonical == "" || protectedHeaders[canonical] {
			continue
		}
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				target.Add(canonical, value)
			}
		}
	}
}
