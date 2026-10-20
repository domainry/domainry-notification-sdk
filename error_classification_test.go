package notificationsdk

import (
	"testing"

	"github.com/domainry/domainry-foundation/apperror"
)

func TestErrorClassificationContract(t *testing.T) {
	tests := []struct {
		status int
		kind   apperror.ErrorKind
		code   string
	}{
		{400, apperror.KindBadRequest, "notification.validation_failed"},
		{403, apperror.KindForbidden, "notification.permission_denied"},
		{409, apperror.KindConflict, "notification.publication_fence_rejected"},
		{503, apperror.KindUnavailable, "notification.database_unavailable"},
	}
	for _, test := range tests {
		err := &Error{StatusCode: test.status, Code: test.code}
		if apperror.KindOf(err) != test.kind || apperror.CodeOf(err) != test.code {
			t.Fatalf("status=%d kind=%q code=%q", test.status, apperror.KindOf(err), apperror.CodeOf(err))
		}
	}
}
