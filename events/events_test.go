package events

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventComparison(t *testing.T) {
	e1 := &Event{
		Summary:     "my meeting",
		Description: "this is my meeting",
		Start:       mustParseTime(t, "2022-05-12T12:00:00Z"),
		End:         mustParseTime(t, "2022-05-12T12:30:00Z"),
		Reminders: []*Reminder{
			{Before: time.Hour, Notified: true},
			{Before: time.Minute, Notified: false},
		},
	}
	e2 := &Event{
		Summary:     "my meeting",
		Description: "this is my meeting",
		Start:       mustParseTime(t, "2022-05-12T12:00:00Z"),
		End:         mustParseTime(t, "2022-05-12T12:30:00Z"),
		Reminders: []*Reminder{
			{Before: time.Hour, Notified: false},
			{Before: time.Minute, Notified: false},
		},
	}

	assert.True(t, cmp.Equal(e1, e2, eventCompareOption),
		cmp.Diff(e1, e2, eventCompareOption))

	e2.Start = mustParseTime(t, "2022-05-12T12:15:00Z")
	assert.False(t, cmp.Equal(e1, e2, eventCompareOption),
		cmp.Diff(e1, e2, eventCompareOption))
}

func mustParseTime(t *testing.T, s string) time.Time {
	ts, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return ts
}
