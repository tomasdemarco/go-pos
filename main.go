package main

import (
	"gitlab.com/g6604/adquirencia/desarrollo/golang_package/iso8583/packager"
	"gitlab.com/g6604/adquirencia/desarrollo/golang_package/iso8583/utils"
	"go-pos/logger"
	"go-pos/server"
	"log"
)

func main() {
	pkg, err := packager.LoadPackager("./iso8583/packager", "iso87BPackager.json")
	if err != nil {
		log.Fatalf("error load packager - %s", err.Error())
	}

	// Logs without flags
	log.SetFlags(0)

	stan := utils.NewStan()

	srv := server.New("server-amex", 1234, pkg, stan, logger.Logger{Level: logger.Info, ErrorDetail: true})
	srv.SetHandler(server.HandleRequest)
	srv.Run()
}
