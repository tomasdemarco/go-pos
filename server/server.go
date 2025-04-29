package server

import (
	"bytes"
	"errors"
	"fmt"
	ctx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/footer"
	"github.com/tomasdemarco/go-pos/header"
	"github.com/tomasdemarco/go-pos/logger"
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
	Name             string
	Network          string
	Port             int
	Timeout          time.Duration
	Packager         *packager.Packager
	Stan             *utils.Stan
	Logger           *logger.Logger
	HandlerFunc      func(c *ctx.RequestContext)
	LengthPackFunc   length.PackFunc
	LengthUnpackFunc length.UnpackFunc
	HeaderPackFunc   header.PackFunc
	HeaderUnpackFunc header.UnpackFunc
	FooterPackFunc   footer.PackFunc
	FooterUnpackFunc footer.UnpackFunc

	maxClients int
	sem        chan struct{}
}

type HandlerFunc func(*ctx.RequestContext, *Server)

func New(
	name string,
	port int,
	timeout int,
	packager *packager.Packager,
	logger *logger.Logger,
	handlerFunc HandlerFunc,
) *Server {

	server := Server{
		Name:             name,
		Network:          "tcp",
		Port:             port,
		Timeout:          time.Duration(timeout) * time.Millisecond,
		Packager:         packager,
		Stan:             utils.NewStan(),
		Logger:           logger,
		LengthPackFunc:   length.Pack,
		LengthUnpackFunc: length.Unpack,
		HeaderPackFunc:   header.Pack,
		HeaderUnpackFunc: header.Unpack,
		FooterPackFunc:   footer.Pack,
		FooterUnpackFunc: footer.Unpack,
		maxClients:       2,
		sem:              make(chan struct{}, 2),
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

			select {
			case s.sem <- struct{}{}: // Intenta adquirir el semáforo
				clientCtx := ctx.NewClientContext(conn)
				go s.handleClient(clientCtx)
			default:
				fmt.Println("Límite de conexiones alcanzado, rechazando:", conn.RemoteAddr())
				conn.Close() // Rechazar la conexión
			}
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
		<-s.sem
		if err != nil {
			s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			return
		}
	}()

	for {
		lengthVal, err := s.LengthUnpackFunc(clientCtx.Reader, s.Packager.Prefix)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			}
			break
		}

		s.Logger.Debug(nil, fmt.Sprintf("received a length message: %d", lengthVal), s.Name)

		msgReq := message.NewMessage(s.Packager)

		msgReq.Length = lengthVal
		_, headerLength, err := s.HeaderUnpackFunc(clientCtx.Reader)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			}
			break
		}

		msgRaw := make([]byte, lengthVal-headerLength)
		_, err = clientCtx.Reader.Read(msgRaw)
		if err != nil {
			if err != io.EOF {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", clientCtx.Conn.RemoteAddr().String(), err)), s.Name)
			}
			break
		}

		s.Logger.Debug(nil, fmt.Sprintf("received a message: %x", msgRaw), s.Name)

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

// SendResponse message for the connection to the client
func (s *Server) SendResponse(ctx *ctx.RequestContext, msg *message.Message) error {
	msgRaw, err := msg.Pack()
	if err != nil {
		return err
	}

	headerRaw, headerLength, err := s.HeaderPackFunc(nil)
	footerRaw, footerLength, err := s.FooterPackFunc(nil)

	lengthPacked, err := length.Pack(s.Packager.Prefix, len(msgRaw)+headerLength+footerLength)
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
	buf.Write(footerRaw)

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
