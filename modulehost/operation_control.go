package modulehost

import (
	"context"

	sharedoperation "github.com/domainry/domainry-foundation/operation"
)

type OperationControl = sharedoperation.Control

type OperationControlStore interface {
	GetOperationControl(context.Context, string, string, string) (OperationControl, bool, error)
	PutOperationControl(context.Context, OperationControl, int64) (bool, error)
}
