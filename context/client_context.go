package context

import (
	"bufio"
	"context"
	"github.com/google/uuid"
	"net"
	"time"
)

type ClientContext struct {
	baseCtx context.Context
	data    map[any]any

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

// Deadline reenvía la llamada al contexto base.
func (c *ClientContext) Deadline() (deadline time.Time, ok bool) {
	return c.baseCtx.Deadline()
}

// Done reenvía la llamada al contexto base.
func (c *ClientContext) Done() <-chan struct{} {
	return c.baseCtx.Done()
}

// Err reenvía la llamada al contexto base.
func (c *ClientContext) Err() error {
	return c.baseCtx.Err()
}

// Value intenta obtener el valor de nuestro mapa interno primero,
// si no lo encuentra, lo busca en el contexto base.
func (c *ClientContext) Value(key any) any {
	if val, ok := c.data[key]; ok {
		return val
	}
	return c.baseCtx.Value(key)
}
