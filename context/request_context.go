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
	return &RequestContext{
		Id:        uuid.New(),
		ClientCtx: clientCtx,
		Request:   msgReq,
		StarTime:  time.Now(),
	}
}
