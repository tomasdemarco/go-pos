package context

import (
	"context"
	"github.com/google/uuid"
	"github.com/tomasdemarco/iso8583/message"
	"time"
)

type RequestContext struct {
	baseCtx context.Context
	data    map[any]any

	Id        uuid.UUID
	ClientCtx *ClientContext
	StarTime  time.Time
	EndTime   time.Time
	Request   *message.Message
	Response  *message.Message
}

func NewRequestContext(clientCtx *ClientContext, msgReq *message.Message) *RequestContext {
	c := RequestContext{
		ClientCtx: clientCtx,
		Request:   msgReq,
		StarTime:  time.Now(),
	}

	c.Id = uuid.New()

	return &c
}

func (c *RequestContext) GetId() uuid.UUID {
	return c.Id
}

func (c *RequestContext) Attributes() *Attributes {
	if c == nil || c.ClientCtx == nil {
		return nil
	}

	return &Attributes{"connId": c.ClientCtx.Id.String()}
}

// Deadline reenvía la llamada al contexto base.
func (c *RequestContext) Deadline() (deadline time.Time, ok bool) {
	return c.baseCtx.Deadline()
}

// Done reenvía la llamada al contexto base.
func (c *RequestContext) Done() <-chan struct{} {
	return c.baseCtx.Done()
}

// Err reenvía la llamada al contexto base.
func (c *RequestContext) Err() error {
	return c.baseCtx.Err()
}

// Value intenta obtener el valor de nuestro mapa interno primero,
// si no lo encuentra, lo busca en el contexto base.
func (c *RequestContext) Value(key any) any {
	if val, ok := c.data[key]; ok {
		return val
	}
	return c.baseCtx.Value(key)
}
