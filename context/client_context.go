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
	Writer     *bufio.Writer
	RemoteAddr string
	StarTime   time.Time
	EndTime    time.Time
}

func NewClientContext(conn net.Conn) *ClientContext {
	return &ClientContext{
		Id:       uuid.New(),
		StarTime: time.Now(),
		Conn:     conn,
		Reader:   bufio.NewReader(conn),
		Writer:   bufio.NewWriter(conn),
	}
}
