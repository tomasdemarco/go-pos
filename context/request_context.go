package context

import (
	"github.com/google/uuid"
	"github.com/tomasdemarco/iso8583/message"
	"time"
)

type RequestContext struct {
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
