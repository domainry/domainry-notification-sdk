package remote_test

import (
	"net/http"
	"testing"
)

func remoteTestHandler(t testing.TB, next http.Handler) http.Handler {
	t.Helper()
	return next
}
