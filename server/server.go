package server

import (
	"bufio"
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
)

type Server struct {
	Name        string
	Network     string
	Port        int
	Packager    *packager.Packager
	Stan        *utils.Stan
	Logger      *logger.Logger
	HandlerFunc func(c *ctx.Context)
}

type HandlerFunc func(*ctx.Context, *Server)

func New(name string, port int, packager *packager.Packager, logger *logger.Logger, handlerFunc HandlerFunc) *Server {
	server := Server{
		Name:     name,
		Network:  "tcp",
		Port:     port,
		Packager: packager,
		Stan:     utils.NewStan(),
		Logger:   logger,
	}

	server.HandlerFunc = func(c *ctx.Context) {
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

			go s.handleClient(conn)
		}
	}
}

// Maneja los clientes que se conectan al switch
func (s *Server) handleClient(conn net.Conn) {
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
		s.Logger.Info(nil, logger.Message, fmt.Sprintf("disconnection to %s", conn.RemoteAddr().String()), s.Name)
		conn.Close()
	}()

	bufReader := bufio.NewReader(conn)
	length, err := message.GetLength(bufReader)
	if err != nil {
		if err != io.EOF {
			return
		}

		s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v. server-%s", conn.RemoteAddr().String(), err, s.Name)))
	}

	recvBuf := make([]byte, length)
	n, err := bufReader.Read(recvBuf)
	for err == nil {
		b := recvBuf[:n]
		messageRaw := fmt.Sprintf("%x", b)

		if len(messageRaw) > s.Packager.PrefixLength+s.Packager.HeaderLength {
			messageGp := message.NewMessage(s.Packager)

			length, err = message.UnpackLength(b)
			if err != nil {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error client %s: %v", conn.RemoteAddr().String(), err)), s.Name)
			}
			messageGp.Length = length

			messageGp.Header, err = message.UnpackHeader(messageRaw, s.Packager)
			if err != nil {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error client %s: %v", conn.RemoteAddr().String(), err)), s.Name)
			} else {
				err = messageGp.Unpack(messageRaw[s.Packager.HeaderLength:])
				if err != nil {
					s.Logger.Error(nil, errors.New(fmt.Sprintf("error client %s: %v", conn.RemoteAddr().String(), err)), s.Name)
				} else {

					c := ctx.New(s.Stan)
					c.Conn = conn
					c.Request = messageGp

					s.Logger.Info(c, logger.IsoUnpack, messageRaw[s.Packager.PrefixLength+s.Packager.HeaderLength:], s.Name)
					err = s.Logger.ISOMessage(c, messageGp, s.Name)
					if err != nil {
						s.Logger.Error(nil, errors.New(fmt.Sprintf("err: %v", err)), s.Name)
					}

					go s.HandlerFunc(c)
				}
			}
		}

		length, err = message.GetLength(bufReader)
		if err == nil {
			recvBuf = make([]byte, length)
			n, err = bufReader.Read(recvBuf)
		}
	}

	if err != io.EOF {
		s.Logger.Error(nil, errors.New(fmt.Sprintf("error read client %s: %v", conn.RemoteAddr().String(), err)), s.Name)
	}
}
