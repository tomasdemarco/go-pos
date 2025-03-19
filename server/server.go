package server

import (
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
)

type Server struct {
	Name             string
	Network          string
	Port             int
	Packager         *packager.Packager
	Stan             *utils.Stan
	Logger           *logger.Logger
	HandlerFunc      func(c *ctx.RequestContext)
	HeaderPackFunc   func(interface{}) (valueRaw []byte, err error)
	HeaderUnpackFunc func(r io.Reader) (value interface{}, length int, err error)
	maxClients       int
	sem              chan struct{}
}

type HandlerFunc func(*ctx.RequestContext, *Server)
type HeaderPackFunc func(interface{}) (valueRaw []byte, err error)
type HeaderUnpackFunc func(r io.Reader) (value interface{}, length int, err error)

func New(
	name string,
	port int,
	packager *packager.Packager,
	logger *logger.Logger,
	handlerFunc HandlerFunc,
	headerPackFunc HeaderPackFunc,
	headerUnpackFunc HeaderUnpackFunc,
) *Server {

	server := Server{
		Name:             name,
		Network:          "tcp",
		Port:             port,
		Packager:         packager,
		Stan:             utils.NewStan(),
		Logger:           logger,
		HeaderPackFunc:   headerPackFunc,
		HeaderUnpackFunc: headerUnpackFunc,
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
		listener.Close()
		s.Logger.Info(nil, logger.Message, fmt.Sprintf("finish listen on port %d", s.Port), s.Name)
	}()

	return nil
}

// Realiza el accept a cada cliente que intenta conectarse
func (s *Server) listenClient(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection refused to %s. server-%s", conn.RemoteAddr().String(), s.Name))
			s.Logger.Error(nil, errors.New(fmt.Sprintf("err accept: %v", err)), s.Name)
		} else {
			s.Logger.Info(nil, logger.Message, fmt.Sprintf("connection established to %s", conn.RemoteAddr().String()), s.Name)
			s.Logger.Info(nil, logger.Message, fmt.Sprintf("accept local port %s / remote host %s", conn.LocalAddr().String(), conn.RemoteAddr().String()), s.Name)

			clientCtx := ctx.NewClientContext(conn)

			go s.handleClient(clientCtx)
		}
	}
}

// Maneja los clientes que se conectan al switch
func (s *Server) handleClient(clientCtx *ctx.ClientContext) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error server-%s: err %v", s.Name, err)))
				s.Logger.Panic(nil, errors.New(fmt.Sprintf("error server-%s: err %v", s.Name, err)), debug.Stack())
			}
		}
	}()

	//Cierra la conexion con el cliente al retornar
	defer func() {
		s.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", clientCtx.Conn.RemoteAddr().String()), s.Name)
		err := clientCtx.Conn.Close()
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			return
		}
	}()

	for {
		length, err := length2.Unpack(clientCtx.Reader, s.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			}
			break
		}

		s.Logger.Debug(nil, fmt.Sprintf("received a length message: %d", length), s.Name)

		msgReq := message.NewMessage(s.Packager)

		msgReq.Length = length
		_, headerLength, err := s.HeaderUnpackFunc(clientCtx.Reader)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			}
			break
		}

		msgRaw := make([]byte, length-headerLength)
		_, err = clientCtx.Reader.Read(msgRaw)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			}
			break
		}

		s.Logger.Debug(nil, fmt.Sprintf("received a message, : %x", msgRaw), s.Name)

		err = msgReq.Unpack(msgRaw)
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
		} else {
			c := ctx.NewRequestContext(clientCtx, msgReq)

			s.Logger.Info(c, logger.IsoUnpack, fmt.Sprintf("%x", msgRaw), s.Name)
			err = s.Logger.ISOMessage(c, msgReq, s.Name)
			if err != nil {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("err: %v", err)), s.Name)
			}

			go s.HandlerFunc(c)
		}
	}
}
