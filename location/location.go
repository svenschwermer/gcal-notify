package location

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/svenschwermer/gcal-notify/config"
	"github.com/svenschwermer/gcal-notify/slack"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

type Slack interface {
	SetWorkingLocation(context.Context, slack.WorkingLocation) error
}

type Bot struct {
	svc   *calendar.Service
	calID string

	slack Slack
}

func NewBot(svc *calendar.Service, calendarID string, slack Slack) *Bot {
	b := &Bot{
		svc:   svc,
		calID: calendarID,
		slack: slack,
	}
	return b
}

func (b *Bot) Poll(ctx context.Context) error {
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ticker.C:
			ticker = time.NewTimer(config.Cfg.LocationPollInterval.D)
		case <-ctx.Done():
			return ctx.Err()
		}

		now := time.Now()
		timeMax := now.Add(24 * time.Hour)
		events, err := b.svc.Events.List(config.Cfg.CalendarID).Context(ctx).EventTypes("workingLocation").Do(
			googleapi.QueryParameter("timeMin", now.Format(time.RFC3339)),
			googleapi.QueryParameter("timeMax", timeMax.Format(time.RFC3339)),
		)
		if err != nil {
			log.Printf("Failed to query event list for location: %v", err)
			continue
		}

		for _, event := range events.Items {
			start, err := time.Parse(time.DateOnly, event.Start.Date)
			if err != nil {
				log.Printf("Failed to parse start time for location: %v", err)
				continue
			}
			end, err := time.Parse(time.DateOnly, event.End.Date)
			if err != nil {
				log.Printf("Failed to parse end time for location: %v", err)
				continue
			}
			if now.Before(start) || end.Before(now) {
				// not applicable
				config.Debug.Printf("Ignoring working location [start=%s end=%s]: %+v",
					event.Start.Date, event.End.Date, event.WorkingLocationProperties)
				continue
			}

			config.Debug.Printf("Setting working location: %+v", event.WorkingLocationProperties)
			switch event.WorkingLocationProperties.Type {
			case "homeOffice":
				err = b.slack.SetWorkingLocation(ctx, slack.WorkingLocationHome)
			case "officeLocation":
				err = b.slack.SetWorkingLocation(ctx, slack.WorkingLocationOffice)
			default:
				err = fmt.Errorf("unsupported working location: %q", event.WorkingLocationProperties.Type)
			}
			if err != nil {
				log.Printf("Failed to set working location: %v", err)
			}

			// we only deal with one matching working location event
			break
		}
	}
}
