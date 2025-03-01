package main

import (
	"github.com/tomasdemarco/iso8583/encoding"
	"github.com/tomasdemarco/iso8583/packager"
	"github.com/tomasdemarco/iso8583/prefix"
	"go-pos/client"
	"go-pos/logger"
	"log"
)

func main() {
	pkg, err := packager.LoadFromJson("./iso8583/packager", "iso87BPackager.json")
	if err != nil {
		log.Fatalf("error load packager - %s", err.Error())
	}

	host := "10.72.0.22"
	port := 8015

	cli := client.New(
		"client-prueba",
		host,
		port,
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

	cli.Listen()

	//srv := server.New(
	//	"server-prueba",
	//	port,
	//	&pkg,
	//	logger.New(
	//		logger.Info,
	//		true,
	//	),
	//	examples.HandleRequest,
	//)
	//
	//err = srv.Run()
	//if err != nil {
	//	log.Fatalf("error running server on port %d: %v", port, err)
	//}
}

func addPackager() *packager.Packager {
	pkg := packager.Packager{
		Name:           "",
		PrefixLength:   4,
		PrefixEncoding: encoding.Hex,
		Fields:         make(map[string]packager.Field),
	}

	fields := make(map[string]packager.Field)
	fields["000"] = packager.Field{
		Length:   4,
		Encoding: encoding.Hex,
		Prefix:   prefix.Prefix{Type: prefix.Fixed, Encoding: encoding.Bcd},
	}

	pkg.Fields = fields

	return &pkg
}
