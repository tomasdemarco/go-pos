package context

import (
	"github.com/google/uuid"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/utils"
	"net"
)

type Context struct {
	Id       uuid.UUID
	Conn     net.Conn
	Stan     int
	Request  *message.Message
	Response *message.Message
}

func New(stan *utils.Stan) *Context {
	return &Context{
		Id:   uuid.New(),
		Stan: stan.Next(),
	}
}
