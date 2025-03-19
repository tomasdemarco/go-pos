package main

import (
	"fmt"
	"github.com/tomasdemarco/go-pos/client"
	reqCtx "github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/logger"
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

	host := "127.0.0.1"
	port := 8015
	//host := "10.72.0.22"
	//port := 8015

	cli := client.New(
		"client-prueba",
		host,
		port,
		20000,
		&pkg,
		&[]string{"007", "011"},
		logger.New(logger.Debug),
		HeaderPackC,
		HeaderUnpackC,
	)

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
			cli.Logger.Error(ctx, err, cli.Name)
		}

		sleepArr := []int{0, 10000, 5000}

		go func() {
			defer wg.Done()

			time.Sleep(time.Duration(sleepArr[0]) * time.Millisecond)
			_, err = cli.Wait(ctx)
			if err != nil {
				cli.Logger.Error(ctx, err, cli.Name)
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

	//header := make(map[string]string)
	//header["01"] = "60"
	//header["02"] = "0001"
	//header["03"] = "0000"
	//msg.Header = header

	msg.SetField("000", "0200")
	//msg.SetField("002", "4123220000000222")
	msg.SetField("003", "000000")
	msg.SetField("004", "000000017078")
	msg.SetField("007", "0227152417")
	msg.SetField("011", fmt.Sprintf("%06d", c.Stan.Next()))
	msg.SetField("012", "152417")
	msg.SetField("013", "0227")
	msg.SetField("014", "3112")
	msg.SetField("022", "900")
	msg.SetField("024", "012")
	msg.SetField("025", "00")
	msg.SetField("035", "4761730000000144D311220118473411")
	msg.SetField("041", "1")
	msg.SetField("042", "1501")
	msg.SetField("048", "001")
	msg.SetField("049", "032")
	//msg.SetField("055", "123")
	//de59 := toISO88591("")
	//msg.SetField("059", "02100010010707")
	msg.SetField("059", "02100010010701029000100107910680008008097C1049AAK0010987009166GP *GPcom014167San Martin 50500416873720041697128004173CABA001174C")
	msg.SetField("060", "GP")
	msg.SetField("062", "0011234")

	return msg
}

//func assembleMessage(ctx *context.Context, c client.Client) *message.Message {
//	msg := message.NewMessage(c.Packager)
//
//	msg.SetField("000", "1100")
//	msg.SetField("002", "341111599241000")
//	msg.SetField("003", "004000")
//	msg.SetField("004", "000000020000")
//	msg.SetField("007", "0227152417")
//	msg.SetField("011", fmt.Sprintf("%06d", ctx.Stan))
//	msg.SetField("012", "250205153740")
//	msg.SetField("014", "2911")
//	msg.SetField("019", "032")
//	msg.SetField("022", "210101W00006")
//	msg.SetField("024", "100")
//	msg.SetField("025", "1900")
//	msg.SetField("026", "8011")
//	msg.SetField("027", "6")
//	msg.SetField("035", "341111599241000=25121011111199911111")
//	msg.SetField("037", "505719003135")
//	msg.SetField("041", "1       ")
//	msg.SetField("042", "7791124928     ")
//	msg.SetField("043", "=GLOBAL PROCESSING QA\\PUAN\\CABA\\C1263AAE  C  032")
//	msg.SetField("049", "032")
//	msg.SetField("053", "1234")
//	msg.SetField("060", "C1E7C1C1C470000000F7F7F9F1F1F2F5F0F1F340404040404040404040F3F7C79396828193D79996A285838995876DD8C1E3858194C79996A4977C93968381934B839694F2F5F4F8F7F9F5F6F2F340404040404040404040")
//
//	return msg
//}

func HeaderUnpackC(r io.Reader) (value interface{}, length int, err error) {

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

func HeaderPackC(interface{}) ([]byte, error) {
	return []byte{0x60, 0x00, 0x00, 0x00, 0x00}, nil
}
