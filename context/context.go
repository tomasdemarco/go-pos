package context

import "github.com/google/uuid"

type Context interface {
	GetId() uuid.UUID
	Attributes() *Attributes
}
