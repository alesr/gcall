package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/alesr/gcall/callback"
	"github.com/alesr/gcall/googlecalendar"
	"github.com/atotto/clipboard"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func main() {
	// Create a logger.
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalln("failed to create logger", err)
	}
	defer logger.Sync()

	// Create a callback server.
	meetingName := flag.String("name", "Instant Meeting", "name of the meeting")
	meetingDuration := flag.Int("duration", 60, "duration of the meeting in minutes")
	flag.Parse()

	codeCh := make(chan string)
	callbackSrv := callback.NewServer(logger, chi.NewRouter(), codeCh)

	// Start the callback server.
	go callbackSrv.Start()

	// Stop the callback server when the program ends.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	defer func() {
		if err := callbackSrv.Stop(ctx); err != nil {
			log.Fatalf("failed to stop callback server: %s", err)
		}
	}()

	// Create a Google Calendar client.
	gCallClient, err := googlecalendar.NewClient(logger, codeCh)
	if err != nil {
		log.Fatalf("failed to create client: %s", err)
	}

	// Create an instant call.
	link, err := gCallClient.CreateInstantCall(*meetingName, time.Duration(*meetingDuration))
	if err != nil {
		log.Fatalf("failed to create instant call: %s", err)
	}

	// Copy the link to the clipboard.
	fmt.Println(link)
	if err := clipboard.WriteAll(link); err != nil {
		log.Fatalf("failed to copy link to clipboard: %s", err)
	}
}
