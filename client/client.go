package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	ctx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/logger"
	length2 "github.com/tomasdemarco/iso8583/length"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
	"github.com/tomasdemarco/iso8583/utils"
	"io"
	"net"
	"runtime/debug"
	"time"
)

type Client struct {
	Name                string
	Network             string
	Host                string
	Port                int
	Timeout             time.Duration
	Conn                *net.TCPConn
	Reader              *bufio.Reader
	Writer              *bufio.Writer
	OngoingTransactions *OngoingTransactions
	Packager            *packager.Packager
	MatchFields         []string
	Stan                *utils.Stan
	Logger              *logger.Logger
	HeaderPackFunc      func(interface{}) (valueRaw []byte, err error)
	HeaderUnpackFunc    func(r io.Reader) (value interface{}, length int, err error)
}

type HandlerFunc func(*ctx.RequestContext, *Client)

func New(
	name string,
	host string,
	port int,
	timeout int,
	packager *packager.Packager,
	matchFields *[]string,
	logger *logger.Logger,
	headerPackFunc func(interface{}) (valueRaw []byte, err error),
	headerUnpackFunc func(r io.Reader) (value interface{}, length int, err error),
) *Client {
	client := Client{
		Name:                name,
		Network:             "tcp",
		Host:                host,
		Port:                port,
		Timeout:             time.Duration(timeout) * time.Millisecond,
		Packager:            packager,
		MatchFields:         []string{"000", "007", "011"},
		Stan:                utils.NewStan(),
		Logger:              logger,
		OngoingTransactions: NewOngoingTransactions(),
		HeaderPackFunc:      headerPackFunc,
		HeaderUnpackFunc:    headerUnpackFunc,
	}

	if matchFields != nil {
		client.MatchFields = *matchFields
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

	c.Conn, err = net.DialTCP(c.Network, nil, tcpAddr)
	if err != nil {
		c.Logger.Info(nil, logger.Message, fmt.Sprintf("connection refused to %s", tcpAddr.String()), c.Name)
		return err
	}

	c.Reader = bufio.NewReader(c.Conn)
	c.Writer = bufio.NewWriter(c.Conn)

	c.Logger.Info(nil, logger.Message, fmt.Sprintf("connection established to %s", tcpAddr.String()), c.Name)

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

	for {
		length, err := length2.Unpack(c.Reader, c.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
			}
			break
		}

		if length <= 0 {
			continue
		}

		msgRes := message.NewMessage(c.Packager)
		msgRes.Length = length
		_, headerLength, err := c.HeaderUnpackFunc(c.Reader)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
			}
			break
		}

		c.Logger.Debug(nil, fmt.Sprintf("received a length message: %d", length), c.Name)

		msgRaw := make([]byte, length-headerLength)
		_, err = c.Reader.Read(msgRaw)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
			}
			break
		}

		c.Logger.Debug(nil, fmt.Sprintf("received a message: %x", msgRaw), c.Name)

		err = msgRes.Unpack(msgRaw)
		if err != nil {
			c.Logger.Error(nil, errors.New(fmt.Sprintf("error server %s: %v", c.Conn.RemoteAddr().String(), err)), c.Name)
		} else {
			var messageId string
			for _, v := range c.MatchFields {
				value, _ := msgRes.GetField(v)
				messageId += value
			}

			if c.OngoingTransactions.List[messageId].Message != nil || !c.OngoingTransactions.IsChanClosed(messageId) {
				c.Logger.Debug(c.OngoingTransactions.List[messageId].Ctx, fmt.Sprintf("received a message, id: %s", messageId), c.Name)
				c.Logger.Info(c.OngoingTransactions.List[messageId].Ctx, logger.IsoUnpack, fmt.Sprintf("%x", msgRaw), c.Name)

				err = c.Logger.ISOMessage(c.OngoingTransactions.List[messageId].Ctx, msgRes, c.Name)
				if err != nil {
					c.Logger.Error(c.OngoingTransactions.List[messageId].Ctx, errors.New(fmt.Sprintf("err: %v", err)), c.Name)
				}

				c.OngoingTransactions.List[messageId].Message <- *msgRes
			} else {
				c.Logger.Debug(nil, fmt.Sprintf("received an unmatched message, id: %s", messageId), c.Name)
				c.Logger.Info(nil, logger.IsoUnpack, fmt.Sprintf("%x", msgRaw), c.Name)

				err = c.Logger.ISOMessage(c.OngoingTransactions.List[messageId].Ctx, msgRes, c.Name)
				if err != nil {
					c.Logger.Error(nil, errors.New(fmt.Sprintf("err: %v", err)), c.Name)
				}
			}
		}
	}
}

// Send message for the connection to the server
func (c *Client) Send(ctx *ctx.RequestContext, msg *message.Message) error {
	messageResponseRaw, err := msg.Pack()
	if err != nil {
		return err
	}

	lengthPacked, err := length2.Pack(c.Packager.Prefix, len(messageResponseRaw)+c.Packager.HeaderLength/2+c.Packager.FooterLength)
	if err != nil {
		return err
	}

	headerResponse, err := c.HeaderPackFunc([]byte{0x60, 0x00, 0x00, 0x00, 0x00})

	c.Logger.Info(ctx, logger.IsoPack, fmt.Sprintf("%x", messageResponseRaw))

	err = c.Logger.ISOMessage(ctx, msg, c.Name)
	if err != nil {
		c.Logger.Error(ctx, err, c.Name)
	}

	var messageId string
	for _, v := range c.MatchFields {
		if v == "000" {
			value, _ := ctx.Request.GetField(v)
			messageId += utils.GetMtiResponse(value)
		} else {
			value, _ := ctx.Request.GetField(v)
			messageId += value
		}
	}

	c.OngoingTransactions.Add(ctx, messageId)

	buf := new(bytes.Buffer)
	buf.Write(lengthPacked)
	buf.Write(headerResponse)
	buf.Write(messageResponseRaw)

	_, err = c.Writer.Write(buf.Bytes())
	if err != nil {
		c.Logger.Error(ctx, err, c.Conn.RemoteAddr().String())
	}

	err = c.Writer.Flush()
	if err != nil {
		c.Logger.Error(ctx, err, c.Conn.RemoteAddr().String())
	}

	c.Logger.Debug(nil, fmt.Sprintf("sent a message: %x", buf.Bytes()), c.Name)

	return nil
}

// Wait for server response
func (c *Client) Wait(reqCtx *ctx.RequestContext) (*message.Message, error) {
	var messageId string
	for _, v := range c.MatchFields {
		if v == "000" {
			value, _ := reqCtx.Request.GetField(v)
			messageId += utils.GetMtiResponse(value)
		} else {
			value, _ := reqCtx.Request.GetField(v)
			messageId += value
		}
	}

	defer c.OngoingTransactions.Remove(messageId)

	select {
	case <-time.After(c.Timeout):
		return nil, errors.New(fmt.Sprintf("transaction %s timeout", messageId))
	case msg := <-c.OngoingTransactions.List[messageId].Message:
		c.Logger.Debug(reqCtx, fmt.Sprintf("received a message channel, id: %s", messageId), c.Name)
		return &msg, nil
	}
}
