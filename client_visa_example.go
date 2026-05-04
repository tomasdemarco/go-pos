package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/tomasdemarco/go-pos/client"
	reqCtx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/iso8583/packager"
	"github.com/tomasdemarco/go-pos/logger"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/prefix"
)

func main() {
	pkg := packager.CreateVisaPackager()

	host := "127.0.0.1"
	port := 8015
	//host := "10.72.0.22"
	//port := 8045

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

	cli.LengthPackFunc = LengthVisaPack
	cli.LengthUnpackFunc = LengthVisaUnpack

	err := cli.Connect()
	if err != nil {
		log.Fatalf("%v", err)
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		wg.Add(1)

		msg := assembleVisaMessage(*cli)

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

func assembleVisaMessage(c client.Client) *message.Message {

	msg := message.NewMessage(c.Packager)

	msg.Header = assembleHeader()

	msg.SetField(0, "0100")
	msg.SetField(2, "4761730000000144")
	msg.SetField(3, "000000")
	msg.SetField(4, "17078")
	msg.SetField(7, "0227152417")
	msg.SetField(11, fmt.Sprintf("%06d", c.Stan.Next()))
	msg.SetField(12, "152417")
	msg.SetField(13, "0227")
	msg.SetField(14, "3112")
	msg.SetField(18, "5533")
	msg.SetField(19, "032")
	msg.SetField(22, "0100")
	msg.SetField(25, "59")
	msg.SetField(32, "446809")
	msg.SetField(37, "531618784394")
	msg.SetField(41, "1")
	msg.SetField(42, "1101")
	msg.SetField(43, "GLOBAL PROCESSING        CABA         AR")
	msg.SetField(60, "000000000740")

	//pkg62, err := packager.LoadFromJson("./iso8583/packager", "subFieldsVisaDe62.json")
	//if err != nil {
	//	log.Fatalf("error load packager - %s", err.Error())
	//}
	//message.RegisterStructField[message.BitmapCustomField](msg, 62)
	//
	//de62 := message.BitmapCustomField{}
	//de62.SubPackager = pkg62
	//de62.SetValue(4, "E")
	//msg.SetField(62, de62)
	//
	//pkg63, err := packager.LoadFromJson("./iso8583/packager", "subFieldsVisaDe63.json")
	//if err != nil {
	//	log.Fatalf("error load packager - %s", err.Error())
	//}
	//message.RegisterStructField[message.BitmapCustomField](msg, 63)
	//
	//de63 := message.BitmapCustomField{}
	//de63.SubPackager = pkg63
	//de63.SetValue(1, "0000")
	//msg.SetField(63, de63)
	//
	//pkg126, err := packager.LoadFromJson("./iso8583/packager", "subFieldsVisaDe126.json")
	//if err != nil {
	//	log.Fatalf("error load packager - %s", err.Error())
	//}
	//message.RegisterStructField[message.BitmapCustomField](msg, 126)

	de126 := packager.NewDE126()
	err := de126.SetFieldString(10, "11 078")
	if err != nil {
		fmt.Println(err)
	}
	//de126Pack, _ := de126.Pack()
	err = msg.SetField(126, de126)
	if err != nil {
		fmt.Println(err)
	}

	return msg
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

func LengthVisaPack(prefixer prefix.Prefixer, lenMessage int) ([]byte, error) {
	b, err := prefixer.EncodeLength(lenMessage)
	if err != nil {
		return nil, err
	}

	return append(b, []byte{0x00, 0x00}...), nil
}

func LengthVisaUnpack(r *bufio.Reader, prefixer prefix.Prefixer) (int, error) {

	buf := make([]byte, prefixer.GetPackedLength()+2)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("reading length: %w", err)
		}

		return 0, err
	}

	result, err := prefixer.DecodeLength(buf[:prefixer.GetPackedLength()], 0)
	if err != nil {
		return 0, err
	}

	return result, err
}

func assembleHeader() *packager.VisaHeader {

	hdr := packager.VisaHeader{}
	hdr.H1 = []byte{0x16}
	hdr.H2 = []byte{0x01}
	hdr.H3 = []byte{0x02}
	hdr.H4 = []byte{0x00, 0xe9}
	hdr.H5 = []byte{0x00, 0x00, 0x00}
	hdr.H6 = []byte{0x86, 0x39, 0x16}
	hdr.H7 = []byte{0x00}
	hdr.H8 = []byte{0x00, 0x00}
	hdr.H9 = []byte{0x00, 0x00, 0x00}
	hdr.H10 = []byte{0x00}
	hdr.H11 = []byte{0x00, 0x00, 0x00}
	hdr.H12 = []byte{0x00}

	return &hdr
}
