package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	ctx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/header"
	"github.com/tomasdemarco/go-pos/logger"
	"github.com/tomasdemarco/go-pos/trailer"
	"github.com/tomasdemarco/iso8583/length"
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
	AutoReconnect       bool
	Conn                *net.TCPConn
	Reader              *bufio.Reader
	Writer              *bufio.Writer
	RemoteAddr          string
	OngoingTransactions *OngoingTransactions
	Packager            *packager.Packager
	MatchFields         []string
	Stan                *utils.Stan
	Logger              *logger.Logger
	LengthPackFunc      length.PackFunc
	LengthUnpackFunc    length.UnpackFunc
	HeaderPackFunc      header.PackFunc
	HeaderUnpackFunc    header.UnpackFunc
	TrailerPackFunc     trailer.PackFunc
	TrailerUnpackFunc   trailer.UnpackFunc
}

type HandlerFunc func(*ctx.RequestContext, *Client)

func New(
	name string,
	host string,
	port int,
	timeout int,
	autoReconnect bool,
	packager *packager.Packager,
	matchFields *[]string,
	logger *logger.Logger,
) *Client {
	client := Client{
		Name:                name,
		Network:             "tcp",
		Host:                host,
		Port:                port,
		Timeout:             time.Duration(timeout) * time.Millisecond,
		AutoReconnect:       autoReconnect,
		Packager:            packager,
		MatchFields:         []string{"000", "007", "011"},
		Stan:                utils.NewStan(),
		Logger:              logger,
		OngoingTransactions: NewOngoingTransactions(),
		LengthPackFunc:      length.Pack,
		LengthUnpackFunc:    length.Unpack,
		HeaderPackFunc:      header.Pack,
		HeaderUnpackFunc:    header.Unpack,
		TrailerPackFunc:     trailer.Pack,
		TrailerUnpackFunc:   trailer.Unpack,
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
	c.RemoteAddr = c.Conn.RemoteAddr().String()

	c.Logger.Info(nil, logger.Message, fmt.Sprintf("connection established to %s", tcpAddr.String()), c.Name)

	go func() {
		c.Listen()

		if c.AutoReconnect {
			err = c.Connect()

			for err != nil {
				time.Sleep(time.Second * 1)
				err = c.Connect()
			}
		}
	}()

	return nil
}

// Disconnect connection to the server
func (c *Client) Disconnect() error {

	if c.Conn != nil {

		err := c.Conn.Close()
		if err != nil {
			return err
		}

		c.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", c.RemoteAddr), c.Name)
	}

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

		c.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", c.RemoteAddr), c.Name)
	}()

	for {
		lengthVal, err := length.Unpack(c.Reader, c.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", c.RemoteAddr, err)), c.Name)
			}
			break
		}

		if lengthVal <= 0 {
			continue
		}

		msgRes := message.NewMessage(c.Packager)
		msgRes.Length = lengthVal
		headerVal, headerLength, err := c.HeaderUnpackFunc(c.Reader)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", c.RemoteAddr, err)), c.Name)
			}
			break
		}

		msgRes.Header = headerVal

		c.Logger.Debug(nil, fmt.Sprintf("received a length message: %d", lengthVal), c.Name)

		msgRaw := make([]byte, lengthVal-headerLength)
		_, err = c.Reader.Read(msgRaw)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", c.RemoteAddr, err)), c.Name)
			}
			break
		}

		c.Logger.Debug(nil, fmt.Sprintf("received a message: %x", msgRaw), c.Name)

		err = msgRes.Unpack(msgRaw)
		if err != nil {
			c.Logger.Error(nil, errors.New(fmt.Sprintf("error server %s: %v", c.RemoteAddr, err)), c.Name)
		} else {
			var messageId string
			for _, v := range c.MatchFields {
				fld, _ := msgRes.GetField(v)
				messageId += fld
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

	headerRaw, headerLength, err := c.HeaderPackFunc(msg.Header)
	trailerRaw, trailerLength, err := c.TrailerPackFunc(msg.Trailer)

	lengthPacked, err := length.Pack(c.Packager.Prefix, len(messageResponseRaw)+headerLength+trailerLength)
	if err != nil {
		return err
	}

	c.Logger.Info(ctx, logger.IsoPack, fmt.Sprintf("%x", messageResponseRaw))

	err = c.Logger.ISOMessage(ctx, msg, c.Name)
	if err != nil {
		c.Logger.Error(ctx, err, c.Name)
		return err
	}

	var messageId string
	for _, v := range c.MatchFields {
		if v == "000" {
			fld, _ := ctx.Request.GetField(v)
			messageId += utils.GetMtiResponse(fld)
		} else {
			fld, _ := ctx.Request.GetField(v)
			messageId += fld
		}
	}

	c.OngoingTransactions.Add(ctx, messageId)

	buf := new(bytes.Buffer)
	buf.Write(lengthPacked)
	buf.Write(headerRaw)
	buf.Write(messageResponseRaw)
	buf.Write(trailerRaw)

	_, err = c.Writer.Write(buf.Bytes())
	if err != nil {
		c.Logger.Error(ctx, err, c.RemoteAddr)
	} else {
		err = c.Writer.Flush()
		if err != nil {
			c.Logger.Error(ctx, err, c.RemoteAddr)
		}
	}

	for err != nil && time.Since(ctx.StarTime) < c.Timeout {
		time.Sleep(time.Second * 1)

		_, err = c.Writer.Write(buf.Bytes())
		if err != nil {
			c.Logger.Error(ctx, err, c.RemoteAddr)
		} else {
			err = c.Writer.Flush()
			if err != nil {
				c.Logger.Error(ctx, err, c.RemoteAddr)
			}
		}
	}

	if err == nil {
		c.Logger.Debug(nil, fmt.Sprintf("sent a message: %x", buf.Bytes()), c.Name)
		return nil
	}

	return err
}

// Wait for server response
func (c *Client) Wait(reqCtx *ctx.RequestContext) (*message.Message, error) {
	var messageId string
	for _, v := range c.MatchFields {
		if v == "000" {
			fld, err := reqCtx.Request.GetField(v)
			if err != nil {
				return nil, err
			}
			messageId += utils.GetMtiResponse(fld)
		} else {
			fld, err := reqCtx.Request.GetField(v)
			if err != nil {
				return nil, err
			}
			messageId += fld
		}
	}

	defer c.OngoingTransactions.Remove(messageId)

	select {
	case <-time.After(c.Timeout - time.Since(reqCtx.StarTime)):
		return nil, errors.New(fmt.Sprintf("transaction %s timeout", messageId))
	case msg := <-c.OngoingTransactions.List[messageId].Message:
		c.Logger.Info(reqCtx, logger.Message, fmt.Sprintf("elapsed time %.3fms", float64(time.Since(reqCtx.StarTime).Nanoseconds())/1e6), c.Name)
		c.Logger.Debug(reqCtx, fmt.Sprintf("received a message channel, id: %s", messageId), c.Name)
		return &msg, nil
	}
}
