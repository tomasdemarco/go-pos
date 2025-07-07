package context

import (
	"bufio"
	"github.com/google/uuid"
	"net"
	"time"
)

type ClientContext struct {
	Id         uuid.UUID
	Conn       net.Conn
	Reader     *bufio.Reader
	Writer     *SafeWriter
	RemoteAddr string
	StarTime   time.Time
	EndTime    time.Time
}

func NewClientContext(conn net.Conn) *ClientContext {
	c := ClientContext{
		StarTime:   time.Now(),
		Conn:       conn,
		Reader:     bufio.NewReader(conn),
		Writer:     NewSafeWriter(conn),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	c.Id = uuid.New()

	return &c
}

func (c *ClientContext) GetId() uuid.UUID {
	return c.Id // Devuelve el campo ID
}

func (c *ClientContext) Attributes() *Attributes {
	if c == nil {
		return nil
	}

	return &Attributes{"connId": c.Id.String()}
}
