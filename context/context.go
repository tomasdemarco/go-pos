package context

import (
	"context"
	"github.com/google/uuid"
)

type Context interface {
	context.Context
	GetId() uuid.UUID
	Attributes() *Attributes
}
