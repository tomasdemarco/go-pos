package server

import (
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

type Server struct {
	Name                 string
	Network              string
	Port                 int
	Packager             *packager.Packager
	Stan                 *utils.Stan
	Logger               *logger.Logger
	HandlerFunc          func(c *ctx.RequestContext)
	LengthPackFunc       length.PackFunc
	LengthUnpackFunc     length.UnpackFunc
	HeaderPackFunc       header.PackFunc
	HeaderUnpackFunc     header.UnpackFunc
	TrailerPackFunc      trailer.PackFunc
	TrailerUnpackFunc    trailer.UnpackFunc
	TrailerGetLengthFunc trailer.GetLengthFunc

	maxClients         int
	sem                chan struct{}
	ReadClientTimeout  time.Duration
	ReadMessageTimeout time.Duration
	MaxMessageSize     int
}

type HandlerFunc func(*ctx.RequestContext, *Server)

type Option func(*Server)

func WithName(name string) Option {
	return func(s *Server) {
		s.Name = name
	}
}

func WithLogger(logger *logger.Logger) Option {
	return func(s *Server) {
		s.Logger = logger
	}
}

func WithMaxClients(max int) Option {
	return func(s *Server) {
		s.maxClients = max
		s.sem = make(chan struct{}, max)
	}
}

func WithReadClientTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.ReadClientTimeout = timeout
	}
}

func WithReadMessageTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.ReadMessageTimeout = timeout
	}
}

func WithMaxMessageSize(size int) Option {
	return func(s *Server) {
		s.MaxMessageSize = size
	}
}

func New(
	port int,
	packager *packager.Packager,
	handlerFunc HandlerFunc,
	opts ...Option,
) *Server {

	// Default values
	server := Server{
		Name:                 "server",
		Network:              "tcp",
		Port:                 port,
		Packager:             packager,
		Stan:                 utils.NewStan(1, 999999),
		Logger:               logger.New(logger.Info, "server"),
		LengthPackFunc:       length.Pack,
		LengthUnpackFunc:     length.Unpack,
		HeaderPackFunc:       header.Pack,
		HeaderUnpackFunc:     header.Unpack,
		TrailerPackFunc:      trailer.Pack,
		TrailerUnpackFunc:    trailer.Unpack,
		TrailerGetLengthFunc: trailer.GetLength,
		maxClients:           10, // Default max clients
		sem:                  make(chan struct{}, 10),
		ReadClientTimeout:    10 * time.Minute,
		ReadMessageTimeout:   10 * time.Second,
		MaxMessageSize:       4096,
	}

	server.HandlerFunc = func(c *ctx.RequestContext) {
		handlerFunc(c, &server)
	}

	// Apply custom options
	for _, opt := range opts {
		opt(&server)
	}

	return &server
}

func (s *Server) Run() error {
	//Inicia a escuchar clientes
	listener, err := net.Listen(s.Network, fmt.Sprintf(":%d", s.Port))
	if err != nil {
		s.Logger.Error(nil, errors.New(fmt.Sprintf("error listening: err %v", err)))
		return err
	}

	s.Logger.Info(nil, logger.Message, fmt.Sprintf("listening on port %d", s.Port))

	//Escucha a los clientes
	s.listenClient(listener)

	//Cierra las conexiones
	defer func() {
		err = listener.Close()
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error finish listen on port %d: %v", s.Port, err)))
		}
		s.Logger.Info(nil, logger.Message, fmt.Sprintf("finish listen on port %d", s.Port))
	}()

	return nil
}

// Realiza el accept a cada cliente que intenta conectarse
func (s *Server) listenClient(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection refused to %s", conn.RemoteAddr().String()))
			s.Logger.Error(nil, errors.New(fmt.Sprintf("err accept: %v", err)))
		} else {
			select {
			case s.sem <- struct{}{}: // Intenta adquirir el semáforo
				clientCtx := ctx.NewClientContext(conn)

				s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection established to %s (%s)", conn.RemoteAddr().String(), clientCtx.Id.String()))
				s.Logger.Info(nil, logger.Message, fmt.Sprintf("accept local port %s / remote host %s (%s)", conn.LocalAddr().String(), conn.RemoteAddr().String(), clientCtx.Id.String()))

				go s.handleClient(clientCtx)
			default:
				s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection limit reached, rejecting: %s", conn.LocalAddr().String()))
				err = conn.Close()
				if err != nil {
					s.Logger.Error(nil, errors.New(fmt.Sprintf("error disconnection client: %v", err)))
				}
			}
		}
	}
}

// Maneja los clientes que se conectan al switch
func (s *Server) handleClient(clientCtx *ctx.ClientContext) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				s.Logger.Error(clientCtx, err)
				s.Logger.Panic(clientCtx, err, debug.Stack())
			}
		}
	}()

	//Cierra la conexion con el cliente al retornar
	defer func() {
		s.Logger.Info(clientCtx, logger.Message, fmt.Sprintf("disconnection to %s", clientCtx.RemoteAddr))
		err := clientCtx.Conn.Close()
		<-s.sem
		if err != nil {
			s.Logger.Error(clientCtx, errors.New(fmt.Sprintf("disconnection to %s: %v", clientCtx.RemoteAddr, err)))
			return
		}
	}()

	for {
		_ = clientCtx.Conn.SetReadDeadline(time.Now().Add(s.ReadClientTimeout))
		lengthVal, err := s.LengthUnpackFunc(clientCtx.Reader, s.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(clientCtx, err)
			}
			break
		}

		if lengthVal == 0 {
			continue
		}

		if lengthVal > s.MaxMessageSize {
			s.Logger.Error(clientCtx, errors.New(fmt.Sprintf("invalid received message length (%d), longer than allowed", lengthVal)))
			return
		}

		msgReq := message.NewMessage(s.Packager)
		c := ctx.NewRequestContext(clientCtx, msgReq)

		s.Logger.Debug(c, fmt.Sprintf("received message length: %d", lengthVal))

		msgReq.Length = lengthVal
		headerVal, headerLength, err := s.HeaderUnpackFunc(clientCtx.Reader)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(c, err)
			}
			break
		}

		msgReq.Header = headerVal

		if msgReq.Header != nil {
			if _, ok := msgReq.Header.([]byte); ok {
				s.Logger.Debug(c, fmt.Sprintf("received message header: %X", msgReq.Header.([]byte)))
			} else {
				s.Logger.Debug(c, fmt.Sprintf("received message header: %v", msgReq.Header))
			}
		}

		_ = clientCtx.Conn.SetReadDeadline(time.Now().Add(s.ReadMessageTimeout))
		msgRaw := make([]byte, lengthVal-headerLength-s.TrailerGetLengthFunc())
		_, err = io.ReadFull(clientCtx.Reader, msgRaw)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(c, err)
			}
			break
		}

		s.Logger.Debug(c, fmt.Sprintf("received a message: %X", msgRaw))

		err = msgReq.Unpack(msgRaw)
		if err != nil {
			s.Logger.Error(c, err)
		} else {

			s.Logger.Info(c, logger.IsoUnpack, fmt.Sprintf("%X", msgRaw))
			s.Logger.Info(c, logger.IsoMessage, msgReq.Log())

			go s.HandlerFunc(c)
		}

		trailerVal, _, err := s.TrailerUnpackFunc(clientCtx.Reader)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(c, err)
			}
			break
		}

		msgReq.Trailer = trailerVal

		if msgReq.Trailer != nil {
			if _, ok := msgReq.Trailer.([]byte); ok {
				s.Logger.Debug(c, fmt.Sprintf("received message trailer: %X", msgReq.Trailer.([]byte)))
			} else {
				s.Logger.Debug(c, fmt.Sprintf("received message trailer: %v", msgReq.Trailer))
			}
		}
	}
}

// SendResponse message for the connection to the client
func (s *Server) SendResponse(ctx *ctx.RequestContext, msg *message.Message) error {
	msgRaw, err := msg.Pack()
	if err != nil {
		return err
	}

	headerRaw, headerLength, err := s.HeaderPackFunc(msg.Header)
	trailerRaw, trailerLength, err := s.TrailerPackFunc(msg.Trailer)

	lengthPacked, err := s.LengthPackFunc(s.Packager.Prefix, len(msgRaw)+headerLength+trailerLength)
	if err != nil {
		return err
	}

	s.Logger.Info(ctx, logger.IsoPack, fmt.Sprintf("%X", msgRaw))
	s.Logger.Info(ctx, logger.IsoMessage, msg.Log())

	buf := new(bytes.Buffer)
	buf.Write(lengthPacked)
	buf.Write(headerRaw)
	buf.Write(msgRaw)
	buf.Write(trailerRaw)

	_, err = ctx.ClientCtx.Writer.Write(buf.Bytes())
	if err != nil {
		return err
	}

	s.Logger.Info(ctx, logger.Message, fmt.Sprintf("elapsed time %.3fms", float64(time.Since(ctx.StarTime).Nanoseconds())/1e6))
	s.Logger.Debug(ctx, fmt.Sprintf("sent a response message: %X", buf.Bytes()))

	return nil
}
