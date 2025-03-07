package main

import (
	"fmt"
	"github.com/tomasdemarco/go-pos/client"
	"github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/logger"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
	"log"
)

func main() {
	pkg, err := packager.LoadFromJson("./iso8583/packager", "iso87EAmexPackager.json")
	if err != nil {
		log.Fatalf("error load packager - %s", err.Error())
	}

	host := "127.0.0.1"
	port := 8015

	cli := client.New(
		"client-prueba",
		host,
		port,
		20000,
		&pkg,
		logger.New(
			logger.Info,
			true,
		),
	)

	err = cli.Connect()
	if err != nil {
		log.Fatalf("%v", err)
	}

	ctx := context.New(cli.Stan)

	msg := assembleMessage(ctx, *cli)

	err = cli.Send(ctx, *msg)
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, err = cli.Wait(ctx, "0227152417000001")
	if err != nil {
		log.Fatalf("%v", err)
	}

	err = cli.Disconnect()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

//func assembleMessage(ctx *context.Context, c client.Client) *message.Message {
//	msg := message.NewMessage(c.Packager)
//
//	header := make(map[string]string)
//	header["01"] = "60"
//	header["02"] = "0001"
//	header["03"] = "0000"
//	msg.Header = header
//
//	msg.SetField("000", "0200")
//	msg.SetField("002", "6502720017344840")
//	msg.SetField("003", "000000")
//	msg.SetField("004", "000000017078")
//	msg.SetField("007", "0227152417")
//	msg.SetField("011", fmt.Sprintf("%06d", ctx.Stan))
//	msg.SetField("012", "152417")
//	msg.SetField("013", "0227")
//	msg.SetField("014", "2911")
//	msg.SetField("022", "810")
//	msg.SetField("024", "012")
//	msg.SetField("025", "00")
//	msg.SetField("041", "1")
//	msg.SetField("042", "1")
//	msg.SetField("048", "001")
//	msg.SetField("049", "032")
//	msg.SetField("055", "123")
//	//de59 := toISO88591("")
//	msg.SetField("059", "02100010010701029000100107910680008008097C1049AAK0010987009166GP *GPcom014167San Martin 5050041687372003169777004173CABA001174C5000005021001GLOBAL PROCESSING S.A012002BUENOS AIRES020003ESTEBAN DE LUCA 2351008004C1049AAK0040055965")
//	msg.SetField("060", "GP")
//	msg.SetField("062", "0011234")
//
//	return msg
//}

func assembleMessage(ctx *context.Context, c client.Client) *message.Message {
	msg := message.NewMessage(c.Packager)

	msg.SetField("000", "1100")
	msg.SetField("002", "341111599241000")
	msg.SetField("003", "004000")
	msg.SetField("004", "000000020000")
	msg.SetField("007", "0227152417")
	msg.SetField("011", fmt.Sprintf("%06d", ctx.Stan))
	msg.SetField("012", "250205153740")
	msg.SetField("014", "2911")
	msg.SetField("019", "032")
	msg.SetField("022", "210101W00006")
	msg.SetField("024", "100")
	msg.SetField("025", "1900")
	msg.SetField("026", "8011")
	msg.SetField("027", "6")
	msg.SetField("035", "341111599241000=25121011111199911111")
	msg.SetField("037", "505719003135")
	msg.SetField("041", "1       ")
	msg.SetField("042", "7791124928     ")
	msg.SetField("043", "=GLOBAL PROCESSING QA\\PUAN\\CABA\\C1263AAE  C  032")
	msg.SetField("049", "032")
	msg.SetField("053", "1234")
	msg.SetField("060", "C1E7C1C1C470000000F7F7F9F1F1F2F5F0F1F340404040404040404040F3F7C79396828193D79996A285838995876DD8C1E3858194C79996A4977C93968381934B839694F2F5F4F8F7F9F5F6F2F340404040404040404040")

	return msg
}
