package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/svenschwermer/gcal-notify/auth"
	"github.com/svenschwermer/gcal-notify/config"
	"github.com/svenschwermer/gcal-notify/events"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var cfgFilePath = flag.String("config", config.DefaultPath, "Configuration file path")

func main() {
	log.SetFlags(0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		cancel()
	}()

	flag.Parse()
	config.Parse(*cfgFilePath)

	switch flag.NArg() {
	case 0:
		doPoll(ctx)
	case 1:
		if flag.Arg(0) == "auth" {
			doAuth(ctx)
		} else {
			log.Fatalf("Unrecognized command %q", flag.Arg(0))
		}
	default:
		log.Fatalf("Expected up to 1 argument, %d provided", flag.NArg())
	}
}

func doAuth(ctx context.Context) {
	ts, err := auth.GetTokenSourceFromWeb(ctx)
	if err != nil {
		log.Fatalf("Failed to get authentication token from web: %v", err)
	}
	if err := writeTokenToDisk(ts); err != nil {
		log.Fatal(err)
	}
}

func doPoll(ctx context.Context) {
	ts, err := auth.GetTokenSourceFromDisk(ctx)
	if err != nil {
		log.Fatalf("Failed to read auth token from disk: %v\nConsider running\n  %s auth",
			err, os.Args[0])
	}
	client := oauth2.NewClient(ctx, ts)
	defer func() {
		if err := writeTokenToDisk(ts); err != nil {
			log.Print(err)
		}
	}()

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	n, err := events.NewNotifier(svc, config.Cfg.CalendarID)
	if err != nil {
		log.Fatalf("Unable to initialize notifier: %v", err)
	}
	n.Poll(ctx)
}

func writeTokenToDisk(ts oauth2.TokenSource) error {
	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("failed to get token for serialization to disk: %w", err)
	} else if err := auth.WriteTokenToDisk(tok); err != nil {
		return fmt.Errorf("failed to write token to disk: %w", err)
	}
	return nil
}
