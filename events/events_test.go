package events

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestEventComparison(t *testing.T) {
	e1 := &Event{
		Summary:     "my meeting",
		Description: "this is my meeting",
		Start:       time.Now(),
		End:         time.Now().Add(time.Hour),
		Reminders: []*Reminder{
			{Before: time.Hour, Notified: true},
			{Before: time.Minute, Notified: false},
		},
	}
	e2 := &Event{
		Summary:     "my meeting",
		Description: "this is my meeting",
		Start:       e1.Start,
		End:         e1.End,
		Reminders: []*Reminder{
			{Before: time.Hour, Notified: false},
			{Before: time.Minute, Notified: false},
		},
	}

	if !cmp.Equal(e1, e2, eventCompareOption) {
		t.Error("expected equality")
		t.Log(cmp.Diff(e1, e2))
	}
}
