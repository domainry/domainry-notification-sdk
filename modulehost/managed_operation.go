package modulehost

import (
	"context"

	sharedoperation "github.com/domainry/domainry-foundation/operation"
	"github.com/domainry/domainry-orm/sqlhost"
)

var ErrManagedOperationIdentityConflict = sharedoperation.ErrIdentityConflict

type OperationScope = sharedoperation.Scope
type OperationCommand = sharedoperation.Command
type ManagedOperation = sharedoperation.ManagedOperation
type ManagedOperationIdentity = sharedoperation.ManagedIdentity
type ManagedOperationQuery = sharedoperation.ManagedQuery
type ManagedOperationTransition = sharedoperation.ManagedTransition
type ManagedOperationClaim = sharedoperation.ManagedClaim

// ManagedOperationStore preserves Notification's worker-facing Claim name
// while the canonical Foundation SQL kernel exposes ClaimManaged to avoid a
// collision with its command-idempotency Claim method.
type ManagedOperationStore interface {
	Create(context.Context, ManagedOperation) error
	Get(context.Context, ManagedOperationIdentity) (ManagedOperation, bool, error)
	List(context.Context, ManagedOperationQuery) ([]ManagedOperation, error)
	Transition(context.Context, ManagedOperationTransition) (ManagedOperation, bool, error)
	Claim(context.Context, ManagedOperationClaim) (ManagedOperation, bool, error)
}

func WithOperationExecutor(ctx context.Context, executor sqlhost.DBTX) context.Context {
	return sharedoperation.WithExecutor(ctx, executor)
}

func OperationExecutorFromContext(ctx context.Context, fallback sqlhost.DBTX) sqlhost.DBTX {
	return sharedoperation.ExecutorFromContext(ctx, fallback)
}
