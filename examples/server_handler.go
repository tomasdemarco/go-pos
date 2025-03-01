package examples

import (
	"fmt"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/utils"
	ctx "go-pos/context"
	"go-pos/logger"
	"go-pos/server"
	"log"
	"math/rand"
	"time"
)

// HandleRequest Handle client request
func HandleRequest(c *ctx.Context, s *server.Server) {
	var messageResponse *message.Message

	mti, err := c.Request.GetField("000")
	if err == nil && mti == "1804" {
		messageResponse = PrepareEchoResponse(c.Request)
	} else {
		messageResponse = PrepareResponse(c.Request)
	}
	messageResponseRaw, _ := messageResponse.Pack()
	lengthHexResponse := message.PackLength(messageResponseRaw, s.Packager.PrefixLength+s.Packager.HeaderLength)
	headerResponse := messageResponse.PackHeader(s.Packager)

	s.Logger.Info(c, logger.IsoPack, messageResponseRaw, s.Name)

	err = s.Logger.ISOMessage(c, messageResponse, s.Name)
	if err != nil {
		log.Printf("error %s: %v", s.Name, err)
	}
	_, err = c.Conn.Write(utils.Hex2Byte(lengthHexResponse + headerResponse + messageResponseRaw))
	if err != nil {
		log.Printf("error %s: %v", s.Name, err)
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
		responseMTI = "0210" // Valor de error o default
	}

	return responseMTI
}
