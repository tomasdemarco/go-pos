package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
	"github.com/tomasdemarco/iso8583/utils"
	ctx "go-pos/context"
	"go-pos/logger"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"time"
)

type Client struct {
	Name                string
	Network             string
	Host                string
	Port                int
	Timeout             time.Duration
	Conn                *net.TCPConn
	OngoingTransactions *OngoingTransactions
	Packager            *packager.Packager
	Stan                *utils.Stan
	Logger              *logger.Logger
}

type HandlerFunc func(*ctx.Context, *Client)

func New(name string, host string, port int, timeout int, packager *packager.Packager, logger *logger.Logger) *Client {
	client := Client{
		Name:     name,
		Network:  "tcp",
		Host:     host,
		Port:     port,
		Timeout:  time.Duration(timeout) * time.Millisecond,
		Packager: packager,
		Stan:     utils.NewStan(),
		Logger:   logger,
		OngoingTransactions: &OngoingTransactions{
			List: make(map[string]OngoingTransaction),
			mu:   &sync.RWMutex{},
		},
	}

	return &client
}

// Connect establishes connection to the server
func (c *Client) Connect() error {
	tcpAddr, err := net.ResolveTCPAddr(c.Network, fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		c.Logger.Error(nil, errors.New(fmt.Sprintf("error connect: err %v", err)), c.Name)
		return err
	}

	conn, err := net.DialTCP(c.Network, nil, tcpAddr)
	if err != nil {
		c.Logger.Info(nil, logger.Message, fmt.Sprintf("connection refused to %s", tcpAddr.String()), c.Name)
		return err
	}

	c.Logger.Info(nil, logger.Message, fmt.Sprintf("connection established to %s", tcpAddr.String()), c.Name)

	c.Conn = conn

	go c.Listen()

	return nil
}

// Disconnect connection to the server
func (c *Client) Disconnect() error {

	err := c.Conn.Close()
	if err != nil {
		return err
	}

	c.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", c.Conn.RemoteAddr().String()), c.Name)

	return nil
}

// Listen for connection to the server
func (c *Client) Listen() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error server-%s: err %v", c.Name, err)))
				c.Logger.Panic(nil, errors.New(fmt.Sprintf("error server-%s: err %v", c.Name, err)), debug.Stack())
			}
		}
	}()

	//Cierra la conexion con el cliente al retornar
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			return
		}

		c.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", c.Conn.RemoteAddr().String()), c.Name)
	}()

	bufReader := bufio.NewReader(c.Conn)
	length, prefixLength, err := message.GetLength(bufReader, c.Packager.Prefix)
	if err != nil {
		if err != io.EOF {
			return
		}

		c.Logger.Error(nil, errors.New(fmt.Sprintf("error read server %s: %v. server-%s", c.Conn.RemoteAddr().String(), err, c.Name)))
	}

	recvBuf := make([]byte, length)
	n, err := bufReader.Read(recvBuf)
	for err == nil {
		b := recvBuf[:n]
		messageRaw := fmt.Sprintf("%x", b)

		if len(messageRaw) > prefixLength+c.Packager.HeaderLength {
			msgResponse := message.NewMessage(c.Packager)

			length, err = message.UnpackLength(b)
			if err != nil {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error server %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
			}
			msgResponse.Length = length

			msgResponse.Header, err = message.UnpackHeader(messageRaw, c.Packager)
			if err != nil {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error server %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
			} else {
				err = msgResponse.Unpack(b[c.Packager.HeaderLength/2:])
				if err != nil {
					c.Logger.Error(nil, errors.New(fmt.Sprintf("error server %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
				} else {

					ct := ctx.New(c.Stan)
					//c.Response = msgResponse

					c.Logger.Info(ct, logger.IsoUnpack, messageRaw[c.Packager.HeaderLength:], c.Name)
					err = c.Logger.ISOMessage(ct, msgResponse, c.Name)
					if err != nil {
						c.Logger.Error(nil, errors.New(fmt.Sprintf("err: %v", err)), c.Name)
					}

					date, _ := msgResponse.GetField("007")
					trace, _ := msgResponse.GetField("011")
					messageId := date + trace

					if c.OngoingTransactions.List[messageId].Message != nil || !IsClosed(c.OngoingTransactions.List[messageId].Message) {
						c.OngoingTransactions.List[messageId].Message <- *msgResponse
					}
				}
			}
		}

		length, prefixLength, err = message.GetLength(bufReader, c.Packager.Prefix)
		if err == nil {
			recvBuf = make([]byte, length)
			n, err = bufReader.Read(recvBuf)
		}
	}
}

func IsClosed(ch <-chan message.Message) bool {
	select {
	case <-ch:
		return true
	default:
	}

	return false
}

// Send message for the connection to the server
func (c *Client) Send(ctx *ctx.Context, msg message.Message) error {
	messageResponseRaw, err := msg.Pack()
	if err != nil {
		return err
	}

	lengthPacked, err := message.PackLength(c.Packager.Prefix, len(messageResponseRaw)+c.Packager.HeaderLength)
	if err != nil {
		return err
	}

	headerResponse := msg.PackHeader(c.Packager)

	c.Logger.Info(ctx, logger.IsoPack, fmt.Sprintf("%x", messageResponseRaw))

	err = c.Logger.ISOMessage(ctx, &msg, c.Name)
	if err != nil {
		c.Logger.Error(ctx, err, c.Name)
	}

	buf := new(bytes.Buffer)
	buf.Write(lengthPacked)
	buf.Write(utils.Hex2Byte(headerResponse))
	buf.Write(messageResponseRaw)

	_, err = c.Conn.Write(buf.Bytes())
	if err != nil {
		c.Logger.Error(ctx, err, c.Conn.RemoteAddr().String())
	}

	date, _ := msg.GetField("007")
	trace, _ := msg.GetField("011")
	messageId := date + trace

	c.OngoingTransactions.Add(messageId, &ctx.Id)

	return nil
}

// Wait for server response
func (c *Client) Wait(ctx *ctx.Context, id string) (*message.Message, error) {
	defer c.OngoingTransactions.Remove(id)

	select {
	case <-time.After(c.Timeout):
		return nil, errors.New("transaction timout")
	case msg := <-c.OngoingTransactions.List[id].Message:
		return &msg, nil
	}
}
