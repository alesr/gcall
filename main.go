package main

import (
	"fmt"
	"log"

	"github.com/alesr/gcall/callback"
	"github.com/alesr/gcall/googlecalendar"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalln("failed to create logger", err)
	}
	defer logger.Sync()

	codeCh := make(chan string)

	callbackSrv := callback.NewServer(logger, chi.NewRouter(), codeCh)
	go callbackSrv.Start()

	defer func() {
		if err := callbackSrv.Stop(); err != nil {
			log.Fatalf("could not stop callback server: %s", err)
		}
	}()

	gCallClient, err := googlecalendar.NewClient(logger, codeCh)
	if err != nil {
		log.Fatalf("could not create client: %s", err)
	}

	link, err := gCallClient.CreateInstantCall()
	if err != nil {
		log.Fatalf("could not create instant call: %s", err)
	}
	fmt.Println(link)
}
