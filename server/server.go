package server

import (
	"bufio"
	"errors"
	"fmt"
	"gitlab.com/g6604/adquirencia/desarrollo/golang_package/iso8583/message"
	"gitlab.com/g6604/adquirencia/desarrollo/golang_package/iso8583/packager"
	"gitlab.com/g6604/adquirencia/desarrollo/golang_package/iso8583/utils"
	ctx "go-pos/context"
	"go-pos/logger"
	"io"
	"log"
	"math/rand"
	"net"
	"runtime/debug"
	"time"
)

type Brand struct {
	Name,
	Port string
	Packager   packager.Packager
	ForcedData bool
}

type Server struct {
	Name        string
	Network     string
	Port        int
	Packager    packager.Packager
	Stan        *utils.Stan
	Logger      logger.Logger
	HandlerFunc func(c *ctx.Context)
}

func New(name string, port int, packager packager.Packager, stan *utils.Stan, customLogger logger.Logger) *Server {
	server := Server{
		Name:     name,
		Network:  "tcp",
		Port:     port,
		Packager: packager,
		Stan:     stan,
		Logger:   customLogger,
	}

	return &server
}

func (s *Server) SetHandler(handlerFunc HandlerFunc) {
	s.HandlerFunc = func(c *ctx.Context) {
		handlerFunc(c, s)
	}
}

func (s *Server) Run() error {
	//Inicia a escuchar clientes
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
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
			messageGp := message.NewMessage(&s.Packager)

			length, err = message.UnpackLength(b)
			if err != nil {
				s.Logger.Error(nil, errors.New(fmt.Sprintf("error client %s: %v", conn.RemoteAddr().String(), err)), s.Name)
			}
			messageGp.Length = length

			messageGp.Header, err = message.UnpackHeader(messageRaw, &s.Packager)
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

type HandlerFunc func(*ctx.Context, *Server)

// Maneja los request de los clientes
func HandleRequest(c *ctx.Context, s *Server) {
	var messageResponse *message.Message

	mti, err := c.Request.GetField("000")
	if err == nil && mti == "1804" {
		messageResponse = PrepareEchoResponse(c.Request)
	} else {
		messageResponse = PrepareResponse(c.Request)
	}
	messageResponseRaw, _ := messageResponse.Pack()
	lengthHexResponse := message.PackLength(messageResponseRaw, s.Packager.PrefixLength+s.Packager.HeaderLength)
	headerResponse := messageResponse.PackHeader(&s.Packager)

	s.Logger.Info(c, logger.IsoPack, messageResponseRaw, s.Name)

	err = s.Logger.ISOMessage(c, messageResponse, s.Name)
	if err != nil {
		log.Printf("error server-%s: %v", s.Name, err)
	}
	_, err = c.Conn.Write(utils.Hex2Byte(lengthHexResponse + headerResponse + messageResponseRaw))
	if err != nil {
		log.Printf("error server-%s: %v", s.Name, err)
	}
}

func PrepareResponse(messageRequest *message.Message) *message.Message {
	messageResponse := message.NewMessage(messageRequest.Packager)

	header := make(map[string]string)
	header["01"] = messageRequest.Header["01"]
	header["02"] = messageRequest.Header["03"]
	header["03"] = messageRequest.Header["02"]
	messageResponse.Header = header

	mti, err := messageRequest.GetField("000")
	if err == nil {
		messageResponse.SetField("000", GetMtiResponse(mti))
	}

	for _, value := range messageRequest.Bitmap {
		if value != "000" && value != "001" {

			fieldAux, err := messageRequest.GetField(value)
			if err == nil {
				messageResponse.SetField(value, fieldAux)
			}
		}
	}

	// Generar un número aleatorio entre 0 y 99999
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := r.Intn(100000)
	de38 := fmt.Sprintf("%06d", num)

	messageResponse.SetField("038", de38)

	messageResponse.SetField("039", "00")

	return messageResponse
}

func PrepareEchoResponse(message800 *message.Message) *message.Message {

	message0810 := message.NewMessage(message800.Packager)

	message0810.SetField("000", "1814")
	fieldAux, err := message800.GetField("003")
	if err == nil {
		message0810.SetField("003", fieldAux)
	}
	fieldAux, err = message800.GetField("007")
	if err == nil {
		message0810.SetField("007", fieldAux)
	}
	fieldAux, err = message800.GetField("011")
	if err == nil {
		message0810.SetField("011", fieldAux)
	}
	fieldAux, err = message800.GetField("012")
	if err == nil {
		message0810.SetField("012", fieldAux)
	}
	fieldAux, err = message800.GetField("024")
	if err == nil {
		message0810.SetField("024", fieldAux)
	}

	message0810.SetField("039", "800")

	return message0810
}

func GetMtiResponse(mti string) string {
	var responseMTI string

	switch mti {
	case "0100":
		responseMTI = "0110"
	case "0200":
		responseMTI = "0210"
	case "0400":
		responseMTI = "0410"
	case "0420":
		responseMTI = "0430"
	case "1100":
		responseMTI = "1110"
	case "1420":
		responseMTI = "1430"
	default:
		log.Println("MTI no reconocido:", mti)
		responseMTI = "0110" // Valor de error o default
	}

	return responseMTI
}
