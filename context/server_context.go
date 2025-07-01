package context

import (
	"bufio"
	"github.com/google/uuid"
	"net"
	"time"
)

type ServerContext struct {
	Id         uuid.UUID
	Conn       net.Conn
	Reader     *bufio.Reader
	Writer     *bufio.Writer
	RemoteAddr string
	StarTime   time.Time
	EndTime    time.Time
}

func NewServerContext(conn net.Conn) *ServerContext {
	c := ServerContext{
		StarTime:   time.Now(),
		Conn:       conn,
		Reader:     bufio.NewReader(conn),
		Writer:     bufio.NewWriter(conn),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	c.Id = uuid.New()

	return &c
}

func (c *ServerContext) GetId() uuid.UUID {
	return c.Id // Devuelve el campo ID
}

func (c *ServerContext) Attributes() *Attributes {
	if c == nil {
		return nil
	}

	return &Attributes{"connId": c.Id.String()}
}
