package modulehost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/domainry/domainry-orm/sqlhost"
)

var ErrManagedOperationIdentityConflict = errors.New("managed operation identity conflict")

type OperationScope struct {
	WorkspaceID   string
	SystemPurpose string
	ResourceType  string
	ResourceID    string
}

type OperationCommand struct {
	ID                 string
	Scope              OperationScope
	Owner              string
	Kind               string
	ActionKey          string
	IdempotencyKey     string
	RequestFingerprint string
	RequestedBy        string
	Reason             string
	Reference          string
	StatusURL          string
	CreatedAt          time.Time
}

func (value OperationCommand) Validate() error {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Owner) == "" || strings.TrimSpace(value.Kind) == "" || strings.TrimSpace(value.ActionKey) == "" {
		return fmt.Errorf("managed operation command identity is required")
	}
	if strings.TrimSpace(value.IdempotencyKey) == "" || strings.TrimSpace(value.RequestFingerprint) == "" || strings.TrimSpace(value.Scope.ResourceType) == "" {
		return fmt.Errorf("managed operation command idempotency and resource identity are required")
	}
	if strings.TrimSpace(value.RequestedBy) == "" || strings.TrimSpace(value.Reason) == "" || strings.TrimSpace(value.StatusURL) == "" || value.CreatedAt.IsZero() {
		return fmt.Errorf("managed operation command audit context is required")
	}
	workspace, system := strings.TrimSpace(value.Scope.WorkspaceID) != "", strings.TrimSpace(value.Scope.SystemPurpose) != ""
	if workspace == system {
		return fmt.Errorf("managed operation requires exactly one workspace or system scope")
	}
	return nil
}

type ManagedOperation struct {
	Command        OperationCommand
	Status         string
	Metadata       json.RawMessage
	Result         json.RawMessage
	ErrorCode      string
	NextAction     string
	LeaseOwner     string
	LeaseExpiresAt string
	FencingToken   int64
	UpdatedAt      time.Time
}

func (value ManagedOperation) Validate() error {
	if err := value.Command.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(value.Status) == "" || value.UpdatedAt.IsZero() {
		return fmt.Errorf("managed operation status and update time are required")
	}
	if !validOperationJSONObject(value.Metadata) || !validOperationJSONObject(value.Result) {
		return fmt.Errorf("managed operation metadata and result must be JSON objects")
	}
	return nil
}

type ManagedOperationIdentity struct {
	ID    string
	Scope OperationScope
	Owner string
	Kind  string
}

func (value ManagedOperationIdentity) Validate() error {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Owner) == "" || strings.TrimSpace(value.Kind) == "" {
		return fmt.Errorf("managed operation identity is required")
	}
	workspace, system := strings.TrimSpace(value.Scope.WorkspaceID) != "", strings.TrimSpace(value.Scope.SystemPurpose) != ""
	if workspace == system {
		return fmt.Errorf("managed operation requires exactly one workspace or system scope")
	}
	return nil
}

type ManagedOperationQuery struct {
	Scope              OperationScope
	Owner              string
	Kind               string
	ResourceID         string
	Statuses           []string
	NextActionBefore   string
	LeaseExpiresBefore string
	Limit              int
	OldestFirst        bool
}

func (value ManagedOperationQuery) Validate() error {
	if strings.TrimSpace(value.Owner) == "" || strings.TrimSpace(value.Kind) == "" {
		return fmt.Errorf("managed operation query owner and kind are required")
	}
	workspace, system := strings.TrimSpace(value.Scope.WorkspaceID) != "", strings.TrimSpace(value.Scope.SystemPurpose) != ""
	if workspace == system || value.Limit < 1 || value.Limit > 1000 {
		return fmt.Errorf("managed operation query scope and limit are invalid")
	}
	for _, status := range value.Statuses {
		if strings.TrimSpace(status) == "" {
			return fmt.Errorf("managed operation query status is required")
		}
	}
	return nil
}

type ManagedOperationTransition struct {
	Identity             ManagedOperationIdentity
	ExpectedStatus       string
	ExpectedLeaseOwner   string
	ExpectedFencingToken int64
	Status               string
	Metadata             json.RawMessage
	Result               json.RawMessage
	ErrorCode            string
	NextAction           string
	ClearLease           bool
	UpdatedAt            time.Time
}

func (value ManagedOperationTransition) Validate() error {
	if err := value.Identity.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(value.ExpectedStatus) == "" || strings.TrimSpace(value.Status) == "" || value.UpdatedAt.IsZero() {
		return fmt.Errorf("managed operation transition statuses and update time are required")
	}
	if !validOperationJSONObject(value.Metadata) || !validOperationJSONObject(value.Result) {
		return fmt.Errorf("managed operation transition metadata and result must be JSON objects")
	}
	if (strings.TrimSpace(value.ExpectedLeaseOwner) == "") != (value.ExpectedFencingToken == 0) {
		return fmt.Errorf("managed operation transition lease owner and fencing token must be supplied together")
	}
	return nil
}

type ManagedOperationClaim struct {
	Identity       ManagedOperationIdentity
	DueStatus      string
	ReclaimStatus  string
	Now            string
	Status         string
	LeaseOwner     string
	LeaseExpiresAt string
	UpdatedAt      time.Time
}

func (value ManagedOperationClaim) Validate() error {
	if err := value.Identity.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(value.DueStatus) == "" || strings.TrimSpace(value.ReclaimStatus) == "" || strings.TrimSpace(value.Now) == "" ||
		strings.TrimSpace(value.Status) == "" || strings.TrimSpace(value.LeaseOwner) == "" || strings.TrimSpace(value.LeaseExpiresAt) == "" || value.UpdatedAt.IsZero() {
		return fmt.Errorf("managed operation claim lifecycle and lease are required")
	}
	return nil
}

type ManagedOperationStore interface {
	Create(context.Context, ManagedOperation) error
	Get(context.Context, ManagedOperationIdentity) (ManagedOperation, bool, error)
	List(context.Context, ManagedOperationQuery) ([]ManagedOperation, error)
	Transition(context.Context, ManagedOperationTransition) (ManagedOperation, bool, error)
	Claim(context.Context, ManagedOperationClaim) (ManagedOperation, bool, error)
}

type operationExecutorContextKey struct{}

func WithOperationExecutor(ctx context.Context, executor sqlhost.DBTX) context.Context {
	if executor == nil {
		return ctx
	}
	return context.WithValue(ctx, operationExecutorContextKey{}, executor)
}

func OperationExecutorFromContext(ctx context.Context, fallback sqlhost.DBTX) sqlhost.DBTX {
	if ctx != nil {
		if executor, ok := ctx.Value(operationExecutorContextKey{}).(sqlhost.DBTX); ok && executor != nil {
			return executor
		}
	}
	return fallback
}

func validOperationJSONObject(value json.RawMessage) bool {
	if len(value) == 0 {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(value, &object) == nil && object != nil
}
