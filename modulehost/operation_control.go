package modulehost

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// OperationControl is durable desired state in the installation-wide
// Operation Controls registry. Reference is owner-defined JSON or an opaque
// identifier; Revision provides compare-and-swap fencing across instances.
type OperationControl struct {
	SystemPurpose string
	Kind          string
	Owner         string
	State         string
	Reason        string
	Reference     string
	UpdatedBy     string
	Revision      int64
	UpdatedAt     time.Time
}

func (value OperationControl) Validate() error {
	if strings.TrimSpace(value.SystemPurpose) == "" || strings.TrimSpace(value.Kind) == "" || strings.TrimSpace(value.Owner) == "" {
		return fmt.Errorf("operation control identity is required")
	}
	if strings.TrimSpace(value.State) == "" || strings.TrimSpace(value.Reason) == "" || strings.TrimSpace(value.UpdatedBy) == "" {
		return fmt.Errorf("operation control state and audit context are required")
	}
	if value.Revision < 1 || value.UpdatedAt.IsZero() {
		return fmt.Errorf("operation control revision and update time are required")
	}
	return nil
}

type OperationControlStore interface {
	GetOperationControl(context.Context, string, string, string) (OperationControl, bool, error)
	PutOperationControl(context.Context, OperationControl, int64) (bool, error)
}
