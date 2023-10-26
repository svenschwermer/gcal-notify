package main

import (
	"context"
	"flag"
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

	flag.Parse()
	config.Parse(*cfgFilePath)

	if flag.NArg() > 0 {
		log.Fatalf("Did not expect argument, %d provided", flag.NArg())
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	ts, err := auth.GetTokenSourceFromDisk(ctx)
	if err != nil {
		log.Fatalf("Failed to read auth token from disk: %v\nConsider running\n  %s auth",
			err, os.Args[0])
	}
	client := oauth2.NewClient(ctx, ts)
	defer auth.WriteTokenToDisk(ts, false)

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
