package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/esiqveland/notify"
	"github.com/godbus/dbus/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/svenschwermer/gcal-notify/browser"
	"github.com/svenschwermer/gcal-notify/config"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

type Notifier struct {
	svc      *calendar.Service
	notifier notify.Notifier
	calID    string

	ev    map[string]*Event // key: ID
	evMtx sync.Mutex

	active    map[uint32]*Event // key: notification ID
	activeMtx sync.Mutex

	checkNotifications chan struct{}
}

func NewNotifier(svc *calendar.Service, calendarID string) (*Notifier, error) {
	n := &Notifier{
		svc:                svc,
		calID:              calendarID,
		ev:                 make(map[string]*Event),
		active:             make(map[uint32]*Event),
		checkNotifications: make(chan struct{}, 1),
	}
	sessionBus, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, fmt.Errorf("failed to connection to session dbus: %w", err)
	}
	if err := sessionBus.Auth(nil); err != nil {
		return nil, fmt.Errorf("failed to authenticate against session dbus: %w", err)
	}
	if err := sessionBus.Hello(); err != nil {
		return nil, fmt.Errorf("failed to send hello message to session dbus: %w", err)
	}
	n.notifier, err = notify.New(sessionBus, notify.WithOnAction(n.onAction), notify.WithOnClosed(n.onClosed))
	if err != nil {
		return nil, fmt.Errorf("failed to create notifier: %w", err)
	}
	return n, nil
}

func (n *Notifier) Poll(ctx context.Context) error {
	go n.notifyWorker(ctx)
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ticker.C:
			ticker = time.NewTimer(config.Cfg.PollInterval.D)
		case <-ctx.Done():
			return ctx.Err()
		}

		timeMin := time.Now()
		timeMax := timeMin.Add(config.Cfg.LookaheadInterval.D)
		events, err := n.svc.Events.List(n.calID).Context(ctx).Do(
			googleapi.QueryParameter("timeMin", timeMin.Format(time.RFC3339)),
			googleapi.QueryParameter("timeMax", timeMax.Format(time.RFC3339)),
			googleapi.QueryParameter("singleEvents", "True"),
		)
		if err != nil {
			log.Printf("Failed to query event list: %v", err)
			continue
		}

		getReminders := func(er *calendar.EventReminders) []*Reminder {
			or := er.Overrides
			if er.UseDefault {
				or = events.DefaultReminders
			}
			r := make([]*Reminder, len(or))
			for i := range or {
				r[i] = &Reminder{Before: time.Duration(or[i].Minutes) * time.Minute}
			}
			return r
		}

		n.evMtx.Lock()

		deletedEvents := make(map[string]bool, len(n.ev))
		for id := range n.ev {
			deletedEvents[id] = true
		}

		for _, event := range events.Items {
			id := event.Id
			existingEvent, isExisting := n.ev[id]
			deletedEvents[id] = false

			if event.Status == "cancelled" {
				if isExisting {
					config.Debug.Printf("Event %q cancelled", event.Summary)
					n.closeNotifications(existingEvent)
					delete(n.ev, id)
				}
				continue
			}
			if !attending(event) {
				if isExisting {
					config.Debug.Printf("Not attending event %q", event.Summary)
					n.closeNotifications(existingEvent)
					delete(n.ev, id)
				}
				continue
			}

			e := &Event{
				Summary:     event.Summary,
				Description: event.Description,
				Hangout:     event.HangoutLink,
				Link:        event.HtmlLink,
				Location:    event.Location,
				Reminders:   getReminders(event.Reminders),
			}
			e.Start, err = time.Parse(time.RFC3339, event.Start.DateTime)
			if err != nil {
				log.Printf("Failed to parse Start %+v: %v", event.Start, err)
				continue
			}
			e.End, err = time.Parse(time.RFC3339, event.End.DateTime)
			if err != nil {
				log.Printf("Failed to parse End %+v: %v", event.End, err)
				continue
			}

			if !isExisting {
				config.Debug.Printf("New event: summary=%q start=%v end=%v reminders=%v",
					e.Summary, e.Start, e.End, e.Reminders)
				n.ev[id] = e
			} else if !cmp.Equal(existingEvent, e, eventCompareOption) {
				config.Debug.Printf("Changed event: summary=%q diff:\n%s",
					e.Summary, cmp.Diff(existingEvent, e, eventCompareOption))
				n.closeNotifications(existingEvent)
				n.ev[id] = e
			}
		}

		for id, deleted := range deletedEvents {
			if deleted {
				e := n.ev[id]
				config.Debug.Printf("Event %q deleted", e.Summary)
				n.closeNotifications(e)
				delete(n.ev, id)
			}
		}

		n.evMtx.Unlock()
	}
}

func attending(e *calendar.Event) bool {
	for _, a := range e.Attendees {
		if a.Self && a.ResponseStatus == "declined" {
			return false
		}
	}
	return true
}

func (n *Notifier) notifyWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		n.evMtx.Lock()
		for id, e := range n.ev {
			if e.End.Before(time.Now()) {
				delete(n.ev, id)
				continue
			} else {
				for _, r := range e.Reminders {
					if !r.Notified && time.Until(e.Start) <= r.Before {
						r.NotificationID = n.doNotify(e)
						r.Notified = true
					}
				}
			}
		}
		n.evMtx.Unlock()

		select {
		case <-ticker.C:
		case <-n.checkNotifications:
		case <-ctx.Done():
			return
		}
	}
}

func (n *Notifier) doNotify(e *Event) uint32 {
	not := notify.Notification{
		AppName: "gcal-notify",
		// https://specifications.freedesktop.org/icon-naming-spec/latest/ar01s04.html
		AppIcon: "x-office-calendar",
		Summary: fmt.Sprintf("%s | %s", e.Start.Format("15:04"), e.Summary),
		Body:    e.Description,
		Actions: []notify.Action{
			{Key: "default", Label: "Default"},
		},
		Hints: map[string]dbus.Variant{},
	}
	if e.Hangout != "" {
		not.AppIcon = "camera-web"
	}
	id, err := n.notifier.SendNotification(not)
	if err != nil {
		log.Printf("Failed to send notification via dbus: %v", err)
	} else {
		config.Debug.Printf("Sent notification: summary=%q id=%d", not.Summary, id)
	}

	n.activeMtx.Lock()
	n.active[id] = e
	n.activeMtx.Unlock()
	return id
}

func (n *Notifier) onAction(action *notify.ActionInvokedSignal) {
	config.Debug.Printf("Notification action: key=%s id=%d", action.ActionKey, action.ID)
	n.activeMtx.Lock()
	defer n.activeMtx.Unlock()
	e, ok := n.active[action.ID]
	if ok {
		if e.Hangout != "" {
			browser.Open(e.Hangout)
		} else if e.Link != "" {
			browser.Open(e.Link)
		}
	}
}

func (n *Notifier) onClosed(closer *notify.NotificationClosedSignal) {
	config.Debug.Printf("Notification closed: reason=%v id=%d", closer.Reason, closer.ID)
	n.activeMtx.Lock()
	delete(n.active, closer.ID)
	n.activeMtx.Unlock()
}

func (n *Notifier) closeNotifications(e *Event) {
	n.activeMtx.Lock()
	defer n.activeMtx.Unlock()

	for _, r := range e.Reminders {
		if r.Notified {
			n.notifier.CloseNotification(r.NotificationID)
			delete(n.active, r.NotificationID)
		}
	}

	select {
	case n.checkNotifications <- struct{}{}:
	default:
	}
}

type Reminder struct {
	Before         time.Duration
	Notified       bool
	NotificationID uint32
}

func (r *Reminder) String() string {
	return fmt.Sprintf("{Before:%v Notified:%t}", r.Before, r.Notified)
}

var eventCompareOption = cmp.FilterPath(func(p cmp.Path) bool {
	switch p.String() {
	case "Reminders.Notified", "Reminders.NotificationID":
		return true // ignore
	default:
		return false
	}
}, cmp.Ignore())

type Event struct {
	Summary     string
	Description string
	Start       time.Time
	End         time.Time
	Hangout     string
	Link        string
	Location    string
	Reminders   []*Reminder
}
