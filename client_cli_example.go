package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tomasdemarco/go-pos/client"
	"github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/go-pos/logger"
	"github.com/tomasdemarco/iso8583/length"
	"github.com/tomasdemarco/iso8583/message"
	"github.com/tomasdemarco/iso8583/packager"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Define flags
	file := flag.String("f", "", "json file to build ISO message")
	ayuda := flag.Bool("h", false, "show help")

	// Analiza los flags
	flag.Parse()

	// Muestra la ayuda si se especifica el flag -ayuda
	if *ayuda {
		flag.Usage()
		return
	}

	pkg, err := packager.LoadFromJson("./iso8583/packager", "iso87BPackager.json")
	if err != nil {
		log.Fatalf("error load packager - %s", err.Error())
	}

	//	host := "127.0.0.1"
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
	cli.HeaderPackFunc = HeaderPackCli
	cli.HeaderUnpackFunc = HeaderUnpackCli

	err = cli.Connect()
	if err != nil {
		log.Fatalf("%v", err)
	}

	msg, err := buildMessageByFile(*cli, *file)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ctx := context.NewRequestContext(nil, msg)

	fmt.Println(msg.Bitmap.GetSliceString())
	fmt.Println(msg.Bitmap.ToString())
	err = cli.Send(ctx, msg)
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, err = cli.Wait(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}

	err = cli.Disconnect()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func buildMessageByFile(c client.Client, file string) (*message.Message, error) {
	absPath, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}

	jsonFile, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var fields map[int]string

	err = json.Unmarshal(byteValue, &fields)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	msg := message.NewMessage(c.Packager)

	now := time.Now()
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	msg.SetField(4, fmt.Sprintf("%012d", random.Intn(99999999-100)+100))
	msg.SetField(7, now.Format("0102150405"))
	msg.SetField(11, fmt.Sprintf("%06d", c.Stan.Next()))
	msg.SetField(12, now.Format("150405"))
	msg.SetField(13, now.Format("0102"))

	fmt.Println(fields)
	for k, v := range fields {
		fmt.Println(msg.Fields)
		fmt.Println("SET:", k)
		msg.SetField(k, v)
	}

	jsonData, err := json.Marshal(msg.Fields)
	if err != nil {
		log.Fatalf("failed to marshal JSON: %v", err)
	}
	log.Println(string(jsonData))

	return msg, nil
}

func HeaderUnpackCli(r io.Reader) (value interface{}, length int, err error) {

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

func HeaderPackCli(interface{}) ([]byte, int, error) {
	return []byte{0x60, 0x00, 0x00, 0x00, 0x00}, 5, nil
}
