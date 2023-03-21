package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/alesr/gcall/callback"
	"github.com/alesr/gcall/clipboard"
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

	meetingName := flag.String("name", "Instant Meeting", "name of the meeting")
	meetingDuration := flag.Int("duration", 60, "duration of the meeting in minutes")

	flag.Parse()

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

	link, err := gCallClient.CreateInstantCall(*meetingName, time.Duration(*meetingDuration))
	if err != nil {
		log.Fatalf("could not create instant call: %s", err)
	}

	fmt.Println(link)

	if clipboard.IsPbCopyAvailable() {
		if err := clipboard.Copy(link); err != nil {
			log.Fatalf("could not copy to clipboard: %s", err)
		}
		fmt.Println("Link copied to clipboard! Ctrl+V to paste it")
	}
}
