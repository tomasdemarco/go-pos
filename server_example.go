package main

import (
	"fmt"
	ctx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/logger"
	"github.com/tomasdemarco/go-pos/server"
	"github.com/tomasdemarco/iso8583/length"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
	"io"
	"log"
	"math/rand"
	"time"
)

func main() {
	pkg, err := packager.LoadFromJson("./iso8583/packager", "iso87BPackager.json")
	if err != nil {
		log.Fatalf("error load packager - %s", err.Error())
	}

	port := 8015

	srv := server.New(
		port,
		pkg,
		HandleRequest,
		server.WithName("server-prueba"),
		server.WithLogger(logger.New(logger.Debug, "server-prueba")),
		server.WithMaxClients(10),
	)

	srv.HeaderPackFunc = HeaderPack
	srv.HeaderUnpackFunc = HeaderUnpack
	srv.LengthPackFunc = length.Pack
	srv.LengthUnpackFunc = length.Unpack

	err = srv.Run()
	if err != nil {
		log.Fatalf("error running server on port %d: %v", port, err)
	}
}

//func addPackager() *packager.Packager {
//	pkg := packager.Packager{
//		Name:   "",
//		Fields: make(map[string]packager.Field),
//	}
//
//	fields := make(map[string]packager.Field)
//	fields["000"] = packager.Field{
//		Length:   4,
//		Encoding: encoding.Hex,
//		Prefix:   prefix.Prefix{Type: prefix.Fixed, Encoding: encoding.Bcd},
//	}
//
//	pkg.Fields = fields
//
//	return &pkg
//}

// HandleRequest Handle client request
func HandleRequest(c *ctx.RequestContext, s *server.Server) {
	var msgRes *message.Message

	fld, err := c.Request.GetField(0)
	if err == nil && fld == "1804" {
		msgRes = PrepareEchoResponse(c.Request)
	} else {
		msgRes = PrepareResponse(c.Request)
	}

	err = s.SendResponse(c, msgRes)
	if err != nil {
		s.Logger.Error(c, fmt.Errorf("error trying to send response message to the client: %w", err))
	}
}

func PrepareResponse(messageRequest *message.Message) *message.Message {
	messageResponse := message.NewMessage(messageRequest.Packager)

	fld, err := messageRequest.GetField(0)
	if err == nil {
		messageResponse.SetField(0, GetMtiResponse(fld))
	}

	for _, value := range messageRequest.Bitmap.GetSliceString() {
		if value != 0 && value != 1 {

			fld, err := messageRequest.GetField(value)
			if err == nil {
				messageResponse.SetField(value, fld)
			}
		}
	}

	// Generar un número aleatorio entre 0 y 99999
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := r.Intn(100000)
	de38 := fmt.Sprintf("%06d", num)

	messageResponse.SetField(38, de38)

	messageResponse.SetField(39, "00")

	return messageResponse
}

func PrepareEchoResponse(message800 *message.Message) *message.Message {

	message0810 := message.NewMessage(message800.Packager)

	message0810.SetField(0, "1814")
	fld, err := message800.GetField(3)
	if err == nil {
		message0810.SetField(3, fld)
	}
	fld, err = message800.GetField(7)
	if err == nil {
		message0810.SetField(7, fld)
	}
	fld, err = message800.GetField(11)
	if err == nil {
		message0810.SetField(11, fld)
	}
	fld, err = message800.GetField(12)
	if err == nil {
		message0810.SetField(12, fld)
	}
	fld, err = message800.GetField(24)
	if err == nil {
		message0810.SetField(24, fld)
	}

	message0810.SetField(39, "800")

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
		responseMTI = "0210" // Valor de error o default
	}

	return responseMTI
}

func HeaderUnpack(r io.Reader) (value interface{}, length int, err error) {

	buf := make([]byte, 5)
	_, err = r.Read(buf)
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("reading header: %w", err)
		}

		return nil, 0, err
	}

	//	h.Value = fmt.Sprintf("%x", buf)

	return fmt.Sprintf("%x", buf), 5, nil
}

func HeaderPack(interface{}) ([]byte, int, error) {
	return []byte{0x60, 0x00, 0x00, 0x00, 0x00}, 5, nil
}
