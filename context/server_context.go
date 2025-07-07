package context

import (
	"bufio"
	"github.com/google/uuid"
	"net"
	"sync"
	"time"
)

type ServerContext struct {
	Id         uuid.UUID
	Conn       net.Conn
	Reader     *bufio.Reader
	Writer     *SafeWriter
	RemoteAddr string
	StarTime   time.Time
	EndTime    time.Time
}

func NewServerContext(conn net.Conn) *ServerContext {
	c := ServerContext{
		StarTime:   time.Now(),
		Conn:       conn,
		Reader:     bufio.NewReader(conn),
		Writer:     NewSafeWriter(conn),
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

type SafeWriter struct {
	writer *bufio.Writer
	mu     sync.Mutex
}

func NewSafeWriter(conn net.Conn) *SafeWriter {
	return &SafeWriter{
		writer: bufio.NewWriter(conn),
	}
}

func (sw *SafeWriter) Write(b []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	n, err = sw.writer.Write(b)
	if err != nil {
		return n, err
	}

	err = sw.writer.Flush()
	return n, err
}
