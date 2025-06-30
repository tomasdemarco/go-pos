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

func New(
	name string,
	port int,
	packager *packager.Packager,
	logger *logger.Logger,
	handlerFunc HandlerFunc,
	maxClients int,
) *Server {

	server := Server{
		Name:                 name,
		Network:              "tcp",
		Port:                 port,
		Packager:             packager,
		Stan:                 utils.NewStan(),
		Logger:               logger,
		LengthPackFunc:       length.Pack,
		LengthUnpackFunc:     length.Unpack,
		HeaderPackFunc:       header.Pack,
		HeaderUnpackFunc:     header.Unpack,
		TrailerPackFunc:      trailer.Pack,
		TrailerUnpackFunc:    trailer.Unpack,
		TrailerGetLengthFunc: trailer.GetLength,
		maxClients:           maxClients,
		sem:                  make(chan struct{}, maxClients),
		ReadClientTimeout:    5 * time.Minute,
		ReadMessageTimeout:   5 * time.Second,
		MaxMessageSize:       4096,
	}

	server.HandlerFunc = func(c *ctx.RequestContext) {
		handlerFunc(c, &server)
	}

	return &server
}

func (s *Server) Run() error {
	//Inicia a escuchar clientes
	listener, err := net.Listen(s.Network, fmt.Sprintf(":%d", s.Port))
	if err != nil {
		s.Logger.Error(nil, errors.New(fmt.Sprintf("error listening: err %v", err)), s.Name)
		return err
	}

	s.Logger.Info(nil, logger.Message, fmt.Sprintf("listening on port %d", s.Port), s.Name)

	//Escucha a los clientes
	s.listenClient(listener)

	//Cierra las conexiones
	defer func() {
		err = listener.Close()
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error finish listen on port%d: %v", s.Port, err)), s.Name)
		}
		s.Logger.Info(nil, logger.Message, fmt.Sprintf("finish listen on port %d", s.Port), s.Name)
	}()

	return nil
}

// Realiza el accept a cada cliente que intenta conectarse
func (s *Server) listenClient(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection refused to %s. %s", conn.RemoteAddr().String(), s.Name))
			s.Logger.Error(nil, errors.New(fmt.Sprintf("err accept: %v", err)), s.Name)
		} else {
			select {
			case s.sem <- struct{}{}: // Intenta adquirir el semáforo
				s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection established to %s", conn.RemoteAddr().String()), s.Name)
				s.Logger.Info(nil, logger.Message, fmt.Sprintf("accept local port %s / remote host %s", conn.LocalAddr().String(), conn.RemoteAddr().String()), s.Name)

				clientCtx := ctx.NewClientContext(conn)
				go s.handleClient(clientCtx)
			default:
				s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection limit reached, rejecting: %s", conn.LocalAddr().String()))
				err = conn.Close()
				if err != nil {
					s.Logger.Error(nil, errors.New(fmt.Sprintf("error disconnection client: %v", err)), s.Name)
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
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error %s: err %v", s.Name, err)))
				s.Logger.Panic(nil, errors.New(fmt.Sprintf("error %s: err %v", s.Name, err)), debug.Stack())
			}
		}
	}()

	//Cierra la conexion con el cliente al retornar
	defer func() {
		s.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", clientCtx.RemoteAddr), s.Name)
		err := clientCtx.Conn.Close()
		<-s.sem
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error disconnection client %s: %v", clientCtx.RemoteAddr, err)), s.Name)
			return
		}
	}()

	for {
		_ = clientCtx.Conn.SetReadDeadline(time.Now().Add(s.ReadClientTimeout))
		lengthVal, err := s.LengthUnpackFunc(clientCtx.Reader, s.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.RemoteAddr, err)), s.Name)
			}
			break
		}

		if lengthVal == 0 {
			continue
		}

		if lengthVal > s.MaxMessageSize {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: invalid length (%d), longer than allowed", clientCtx.RemoteAddr, lengthVal)), s.Name)
			return
		}

		s.Logger.Debug(nil, fmt.Sprintf("received message length: %d", lengthVal), s.Name)

		msgReq := message.NewMessage(s.Packager)

		msgReq.Length = lengthVal
		headerVal, headerLength, err := s.HeaderUnpackFunc(clientCtx.Reader)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.RemoteAddr, err)), s.Name)
			}
			break
		}

		msgReq.Header = headerVal

		if msgReq.Header != nil {
			if _, ok := msgReq.Header.([]byte); ok {
				s.Logger.Debug(nil, fmt.Sprintf("received message header: %x", msgReq.Header.([]byte)), s.Name)
			} else {
				s.Logger.Debug(nil, fmt.Sprintf("received message header: %v", msgReq.Header), s.Name)
			}
		}

		_ = clientCtx.Conn.SetReadDeadline(time.Now().Add(s.ReadMessageTimeout))
		msgRaw := make([]byte, lengthVal-headerLength-s.TrailerGetLengthFunc())
		_, err = io.ReadFull(clientCtx.Reader, msgRaw)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.RemoteAddr, err)), s.Name)
			}
			break
		}

		s.Logger.Debug(nil, fmt.Sprintf("received a message: %x", msgRaw), s.Name)

		err = msgReq.Unpack(msgRaw)
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error client %s: %v", clientCtx.RemoteAddr, err)), s.Name)
		} else {
			c := ctx.NewRequestContext(clientCtx, msgReq)

			s.Logger.Info(c, logger.IsoUnpack, fmt.Sprintf("%x", msgRaw), s.Name)
			err = s.Logger.ISOMessage(c, msgReq, s.Name)
			if err != nil {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("err: %v", err)), s.Name)
			}

			go s.HandlerFunc(c)
		}

		trailerVal, _, err := s.TrailerUnpackFunc(clientCtx.Reader)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.RemoteAddr, err)), s.Name)
			}
			break
		}

		msgReq.Trailer = trailerVal

		if msgReq.Trailer != nil {
			if _, ok := msgReq.Trailer.([]byte); ok {
				s.Logger.Debug(nil, fmt.Sprintf("received message trailer: %x", msgReq.Trailer.([]byte)), s.Name)
			} else {
				s.Logger.Debug(nil, fmt.Sprintf("received message trailer: %v", msgReq.Trailer), s.Name)
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

	s.Logger.Info(ctx, logger.IsoPack, fmt.Sprintf("%x", msgRaw))

	err = s.Logger.ISOMessage(ctx, msg, s.Name)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.Write(lengthPacked)
	buf.Write(headerRaw)
	buf.Write(msgRaw)
	buf.Write(trailerRaw)

	_, err = ctx.ClientCtx.Writer.Write(buf.Bytes())
	if err != nil {
		return err
	}

	err = ctx.ClientCtx.Writer.Flush()
	if err != nil {
		return err
	}

	s.Logger.Info(ctx, logger.Message, fmt.Sprintf("elapsed time %.3fms", float64(time.Since(ctx.StarTime).Nanoseconds())/1e6), s.Name)
	s.Logger.Debug(ctx, fmt.Sprintf("sent a response message: %x", buf.Bytes()), s.Name)

	return nil
}
