package main

import (
	"fmt"
	"github.com/tomasdemarco/go-pos/client"
	reqCtx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/logger"
	"github.com/tomasdemarco/iso8583/header"
	"github.com/tomasdemarco/iso8583/length"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
	"io"
	"log"
	"sync"
	"time"
)

func main() {
	pkg, err := packager.LoadFromJson("./iso8583/packager", "iso87BPackager.json")
	if err != nil {
		log.Fatalf("error load packager - %s", err.Error())
	}
	pkg.Header = &HeaderGpPackager{}

	//host := "127.0.0.1"
	//port := 8015
	host := "10.72.0.22"
	port := 8015

	cli := client.New(
		host,
		port,
		pkg,
		client.WithName("client-prueba"),
		client.WithTimeout(30*time.Second),
		client.WithAutoReconnect(true),
		client.WithMatchFields([]int{7, 11}),
		client.WithLogger(logger.New(logger.Debug, "client-prueba")),
	)

	cli.LengthPackFunc = length.Pack
	cli.LengthUnpackFunc = length.Unpack

	err = cli.Connect()
	if err != nil {
		log.Fatalf("%v", err)
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)

		msg := assembleMessage(*cli)

		ctx := reqCtx.NewRequestContext(nil, msg)

		err = cli.Send(ctx, msg)
		if err != nil {
			cli.Logger.Error(ctx, err)
		}

		sleepArr := []int{0, 10000, 5000}

		go func() {
			defer wg.Done()

			time.Sleep(time.Duration(sleepArr[0]) * time.Millisecond)
			_, err = cli.Wait(ctx)
			if err != nil {
				cli.Logger.Error(ctx, err)
			}
		}()

		time.Sleep(time.Duration(2) * time.Millisecond)
	}

	wg.Wait()

	err = cli.Disconnect()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func assembleMessage(c client.Client) *message.Message {

	msg := message.NewMessage(c.Packager)

	msg.Header = assembleHeaderGp()

	msg.SetField(0, "0200")
	msg.SetField(2, "4761730000000144")
	msg.SetField(3, "000000")
	msg.SetField(4, "17078")
	msg.SetField(7, "0227152417")
	msg.SetField(11, fmt.Sprintf("%06d", c.Stan.Next()))
	msg.SetField(12, "152417")
	msg.SetField(13, "0227")
	msg.SetField(14, "3112")
	msg.SetField(22, "051")
	msg.SetField(23, "001")
	msg.SetField(24, "012")
	msg.SetField(25, "00")
	msg.SetField(35, "4761730000000144=311220118473411")
	msg.SetField(41, "1")
	msg.SetField(42, "1101")
	msg.SetField(48, "001")
	msg.SetField(49, "032")
	msg.SetField(55, "9F2608A34B9543C74723EE9F2701809F101706011203A000000F00564953414C3354455354434153459F3704A86CC8E39F36020001950580800080009A032306279C01009F02060000000110005F2A020032820218009F1A0200329F34031E03009F3303E0F8C88407A00000000320109F03060000000000009F350122")
	//de59 := toISO88591("")
	//msg.SetField("059", "02100010010707")
	msg.SetField(59, "021000100107070680008008097C1049AAK0010988020166YAG*GP Abarrotes Jes018167Avda Caseros, 286200416854110041694280009173Balvanera0011741")
	msg.SetField(60, "GP")

	message.RegisterStructField[CustomerInfo](msg, 62)

	originalInfo := CustomerInfo{Ticket: "0123", Lote: "001"}
	msg.SetField(62, originalInfo)

	return msg
}

// CustomerInfo Un struct personalizado que implementa CustomPacker
type CustomerInfo struct {
	Ticket string `json:"ticket"`
	Lote   string `json:"lote"`
}

func (c *CustomerInfo) Pack() (string, error) {
	return fmt.Sprintf("%s%s", c.Ticket, c.Lote), nil
}

func (c *CustomerInfo) Unpack(data string) error {
	if len(data) > 4 {
		return fmt.Errorf("formato inválido para CustomerInfo: se esperaban mas de 4 caracteres, se obtuvieron %d", len(data))
	}
	c.Ticket = data[:4]
	c.Lote = data[4:]
	return nil
}

//func assembleMessage(ctx *context.Context, c client.Client) *message.Message {
//	msg := message.NewMessage(c.Packager)
//
//	msg.SetField(0, "1100")
//	msg.SetField(2, "341111599241000")
//	msg.SetField(3, "004000")
//	msg.SetField(4, "000000020000")
//	msg.SetField(7, "0227152417")
//	msg.SetField(11, fmt.Sprintf("%06d", ctx.Stan))
//	msg.SetField(12, "250205153740")
//	msg.SetField(14, "2911")
//	msg.SetField(19, "032")
//	msg.SetField(22, "210101W00006")
//	msg.SetField(24, "100")
//	msg.SetField(25, "1900")
//	msg.SetField(26, "8011")
//	msg.SetField(27, "6")
//	msg.SetField(35, "341111599241000=25121011111199911111")
//	msg.SetField(37, "505719003135")
//	msg.SetField(41, "1       ")
//	msg.SetField(42, "7791124928     ")
//	msg.SetField(43, "=GLOBAL PROCESSING QA\\PUAN\\CABA\\C1263AAE  C  032")
//	msg.SetField(49, "032")
//	msg.SetField(53, "1234")
//	msg.SetField(60, "C1E7C1C1C470000000F7F7F9F1F1F2F5F0F1F340404040404040404040F3F7C79396828193D79996A285838995876DD8C1E3858194C79996A4977C93968381934B839694F2F5F4F8F7F9F5F6F2F340404040404040404040")
//
//	return msg
//}

type GpHeader struct {
	MessageId     []byte
	SourceId      []byte
	DestinationId []byte
}

func (h *GpHeader) Get() any { return h }
func (h *GpHeader) Set(hdr any) {
	if val, ok := hdr.(*GpHeader); ok {
		h.MessageId = val.MessageId
		h.SourceId = val.SourceId
		h.DestinationId = val.DestinationId
	}
}

func (h *GpHeader) Log() string {
	return fmt.Sprintf("MsgId: %X | SrcId: %X | DstId: %X", h.MessageId, h.SourceId, h.DestinationId)
}

type HeaderGpPackager struct{}

func (h *HeaderGpPackager) Unpack(r io.Reader) (value header.Header, length int, err error) {

	buf := make([]byte, 5)
	_, err = r.Read(buf)
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("reading header: %w", err)
		}

		return nil, 0, err
	}

	return unpackHeaderGp(buf), 5, nil
}

func (h *HeaderGpPackager) Pack(hdr header.Header) ([]byte, int, error) {
	var b []byte
	if hdr, ok := hdr.Get().(*GpHeader); ok {
		b = append(b, hdr.MessageId...)
		b = append(b, hdr.SourceId...)
		b = append(b, hdr.DestinationId...)
	}
	return b, 5, nil
}

func assembleHeaderGp() *GpHeader {

	hdr := GpHeader{}
	hdr.MessageId = []byte{0x60}
	hdr.SourceId = []byte{0x00, 0x01}
	hdr.DestinationId = []byte{0x00, 0x00}

	return &hdr
}

func unpackHeaderGp(b []byte) *GpHeader {

	if len(b) == 5 {
		h := GpHeader{}
		h.MessageId = b[:1]
		h.SourceId = b[1:3]
		h.DestinationId = b[3:5]
		return &h
	}

	return nil
}
