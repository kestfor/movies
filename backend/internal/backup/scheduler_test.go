package backup

import (
	"testing"
	"time"
)

func TestNextRunUsesTodayWhenTimeIsAhead(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("test", 0)
	now := time.Date(2026, 7, 12, 2, 0, 0, 0, loc)
	got := nextRun(now, DailySchedule{Hour: 3, Minute: 0}, loc)
	want := time.Date(2026, 7, 12, 3, 0, 0, 0, loc)

	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNextRunUsesTomorrowWhenTimePassed(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("test", 0)
	now := time.Date(2026, 7, 12, 4, 0, 0, 0, loc)
	got := nextRun(now, DailySchedule{Hour: 3, Minute: 0}, loc)
	want := time.Date(2026, 7, 13, 3, 0, 0, 0, loc)

	if !got.Equal(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}
