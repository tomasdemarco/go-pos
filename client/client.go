package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/tomasdemarco/go-pos/context"
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
	Writer              *context.SafeWriter
	RemoteAddr          string
	OngoingTransactions *OngoingTransactions
	Packager            *packager.Packager
	MatchFields         []int
	Stan                *utils.Stan
	Logger              *logger.Logger
	LengthPackFunc      length.PackFunc
	LengthUnpackFunc    length.UnpackFunc
	HeaderPackFunc      header.PackFunc
	HeaderUnpackFunc    header.UnpackFunc
	TrailerPackFunc     trailer.PackFunc
	TrailerUnpackFunc   trailer.UnpackFunc

	readServerTimeout  time.Duration
	readMessageTimeout time.Duration
	maxMessageSize     int
}

type HandlerFunc func(*context.RequestContext, *Client)

type ClientOption func(*Client)

func WithName(name string) ClientOption {
	return func(c *Client) {
		c.Name = name
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.Timeout = timeout
	}
}

func WithAutoReconnect(autoReconnect bool) ClientOption {
	return func(c *Client) {
		c.AutoReconnect = autoReconnect
	}
}

func WithMatchFields(matchFields []int) ClientOption {
	return func(c *Client) {
		c.MatchFields = matchFields
	}
}

func WithLogger(logger *logger.Logger) ClientOption {
	return func(c *Client) {
		c.Logger = logger
	}
}

func New(
	host string,
	port int,
	packager *packager.Packager,
	opts ...ClientOption,
) *Client {
	client := Client{
		Name:                "client",
		Network:             "tcp",
		Host:                host,
		Port:                port,
		Timeout:             30 * time.Second,
		AutoReconnect:       true,
		Packager:            packager,
		MatchFields:         []int{0, 7, 11},
		Stan:                utils.NewStan(1, 999999),
		Logger:              logger.New(logger.Info, "client"),
		OngoingTransactions: NewOngoingTransactions(),
		LengthPackFunc:      length.Pack,
		LengthUnpackFunc:    length.Unpack,
		HeaderPackFunc:      header.Pack,
		HeaderUnpackFunc:    header.Unpack,
		TrailerPackFunc:     trailer.Pack,
		TrailerUnpackFunc:   trailer.Unpack,
		readServerTimeout:   5 * time.Minute,
		readMessageTimeout:  5 * time.Second,
		maxMessageSize:      4096,
	}

	for _, opt := range opts {
		opt(&client)
	}

	return &client
}

// Connect establishes connection to the server
func (c *Client) Connect() error {
	tcpAddr, err := net.ResolveTCPAddr(c.Network, fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		c.Logger.Error(nil, errors.New(fmt.Sprintf("error connect: %v", err)))
		return err
	}

	c.Conn, err = net.DialTCP(c.Network, nil, tcpAddr)
	if err != nil {
		c.Logger.Info(nil, logger.Message, fmt.Sprintf("connection refused to %s", tcpAddr.String()))
		return err
	}

	serverContext := context.NewServerContext(c.Conn)

	c.Logger.Info(serverContext, logger.Message, fmt.Sprintf("connection established to %s", tcpAddr.String()))
	c.Reader = bufio.NewReader(c.Conn)
	c.Writer = context.NewSafeWriter(c.Conn)
	go func() {
		c.Listen(serverContext)

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

		c.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", c.RemoteAddr))
	}

	return nil
}

// Listen for connection to the server
func (c *Client) Listen(ctx *context.ServerContext) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				c.Logger.Error(ctx, err)
				c.Logger.Panic(ctx, err, debug.Stack())
			}
		}
	}()

	//Cierra la conexion con el cliente al retornar
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			return
		}

		c.Logger.Info(ctx, logger.Message, fmt.Sprintf("disconnection to %s", c.RemoteAddr))
	}()

	for {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.readServerTimeout))
		lengthVal, err := length.Unpack(c.Reader, c.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(ctx, err)
			}
			break
		}

		if lengthVal == 0 {
			continue
		}

		if lengthVal > c.maxMessageSize {
			c.Logger.Error(nil, errors.New(fmt.Sprintf("invalid received message length (%d), longer than allowed", lengthVal)))
			return
		}

		c.Logger.Debug(ctx, fmt.Sprintf("received message length: %d", lengthVal))

		msgRes := message.NewMessage(c.Packager)
		msgRes.Length = lengthVal
		headerVal, headerLength, err := c.HeaderUnpackFunc(c.Reader)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(ctx, err)
			}
			break
		}

		msgRes.Header = headerVal

		if msgRes.Header != nil {
			if _, ok := msgRes.Header.([]byte); ok {
				c.Logger.Debug(ctx, fmt.Sprintf("received message header: %X", msgRes.Header.([]byte)))
			} else {
				c.Logger.Debug(ctx, fmt.Sprintf("received message header: %v", msgRes.Header))
			}
		}

		_ = c.Conn.SetReadDeadline(time.Now().Add(c.readMessageTimeout))
		msgRaw := make([]byte, lengthVal-headerLength)
		_, err = io.ReadFull(c.Reader, msgRaw)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(ctx, err)
			}
			break
		}

		trailerVal, trailerLength, err := c.TrailerUnpackFunc(c.Reader)
		if err != nil {
			if err != io.EOF {
				c.Logger.Error(ctx, err)
			}
			break
		}

		msgRaw = msgRaw[:len(msgRaw)-trailerLength]

		msgRes.Trailer = trailerVal

		if msgRes.Trailer != nil {
			if _, ok := msgRes.Trailer.([]byte); ok {
				c.Logger.Debug(ctx, fmt.Sprintf("received message trailer: %X", msgRes.Trailer.([]byte)))
			} else {
				c.Logger.Debug(ctx, fmt.Sprintf("received message trailer: %v", msgRes.Trailer))
			}
		}

		c.Logger.Debug(ctx, fmt.Sprintf("received a message: %X", msgRaw))

		err = msgRes.Unpack(msgRaw)
		if err != nil {
			c.Logger.Error(ctx, err)
		} else {
			var messageId string
			for _, v := range c.MatchFields {
				fld, _ := msgRes.GetField(v)
				messageId += fld
			}

			if c.OngoingTransactions.List[messageId].Message != nil || !c.OngoingTransactions.IsChanClosed(messageId) {
				c.Logger.Debug(c.OngoingTransactions.List[messageId].Ctx, fmt.Sprintf("received a message, id: %s", messageId))
				c.Logger.Info(c.OngoingTransactions.List[messageId].Ctx, logger.IsoUnpack, fmt.Sprintf("%X", msgRaw))
				c.Logger.Info(c.OngoingTransactions.List[messageId].Ctx, logger.IsoMessage, msgRes.Log())

				c.OngoingTransactions.List[messageId].Message <- *msgRes
			} else {
				c.Logger.Debug(ctx, fmt.Sprintf("received an unmatched message, id: %s", messageId))
				c.Logger.Info(ctx, logger.IsoUnpack, fmt.Sprintf("%X", msgRaw))
				c.Logger.Info(c.OngoingTransactions.List[messageId].Ctx, logger.IsoMessage, msgRes.Log())
			}
		}
	}
}

// Send message for the connection to the server
func (c *Client) Send(ctx *context.RequestContext, msg *message.Message) error {

	fmt.Println(msg.Bitmap.GetSliceString())
	fmt.Println(msg.Bitmap.ToString())
	messageResponseRaw, err := msg.Pack()
	if err != nil {
		return err
	}

	fmt.Println(msg.Bitmap.GetSliceString())
	fmt.Println(msg.Bitmap.ToString())
	headerRaw, headerLength, err := c.HeaderPackFunc(msg.Header)
	trailerRaw, trailerLength, err := c.TrailerPackFunc(msg.Trailer)

	lengthPacked, err := length.Pack(c.Packager.Prefix, len(messageResponseRaw)+headerLength+trailerLength)
	if err != nil {
		return err
	}

	c.Logger.Info(ctx, logger.IsoPack, fmt.Sprintf("%X", messageResponseRaw))
	c.Logger.Info(ctx, logger.IsoMessage, msg.Log())

	var messageId string
	for _, v := range c.MatchFields {
		if v == 0 {
			fld, err := ctx.Request.GetField(v)
			if err != nil {
				return err
			}

			mti, err := utils.GetMtiResponse(fld)
			if err != nil {
				return err
			}
			messageId += mti
		} else {
			fld, err := ctx.Request.GetField(v)
			if err != nil {
				return err
			}
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
		c.Logger.Error(ctx, err)
	}

	for err != nil && time.Since(ctx.StarTime) < c.Timeout {
		time.Sleep(time.Second * 1)

		_, err = c.Writer.Write(buf.Bytes())
		if err != nil {
			c.Logger.Error(ctx, err)
		}
	}

	if err == nil {
		c.Logger.Debug(ctx, fmt.Sprintf("sent a message: %X", buf.Bytes()))
		return nil
	}

	return err
}

// Wait for server response
func (c *Client) Wait(reqCtx *context.RequestContext) (*message.Message, error) {
	var messageId string
	for _, v := range c.MatchFields {
		if v == 0 {
			fld, err := reqCtx.Request.GetField(v)
			if err != nil {
				return nil, err
			}

			mti, err := utils.GetMtiResponse(fld)
			if err != nil {
				return nil, err
			}
			messageId += mti
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
		c.Logger.Info(reqCtx, logger.Message, fmt.Sprintf("elapsed time %.3fms", float64(time.Since(reqCtx.StarTime).Nanoseconds())/1e6))
		c.Logger.Debug(reqCtx, fmt.Sprintf("received a message channel, id: %s", messageId))
		return &msg, nil
	}
}
