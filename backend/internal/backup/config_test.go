package backup

import "testing"

func TestParseDailySchedule(t *testing.T) {
	t.Parallel()

	schedule, err := ParseDailySchedule("03:15")
	if err != nil {
		t.Fatalf("ParseDailySchedule returned error: %v", err)
	}
	if schedule.Hour != 3 || schedule.Minute != 15 {
		t.Fatalf("unexpected schedule: %+v", schedule)
	}
	if schedule.String() != "03:15" {
		t.Fatalf("unexpected schedule string: %s", schedule.String())
	}
}

func TestParseDailyScheduleRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	values := []string{"3", "24:00", "10:60", "aa:00", "10:bb"}
	for _, value := range values {
		if _, err := ParseDailySchedule(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}

func TestParseAdminChatIDs(t *testing.T) {
	t.Parallel()

	chatIDs, err := parseAdminChatIDs("123, -456,789")
	if err != nil {
		t.Fatalf("parseAdminChatIDs returned error: %v", err)
	}
	want := []int64{123, -456, 789}
	if len(chatIDs) != len(want) {
		t.Fatalf("unexpected chat ids: %+v", chatIDs)
	}
	for i := range want {
		if chatIDs[i] != want[i] {
			t.Fatalf("chat id %d: got %d, want %d", i, chatIDs[i], want[i])
		}
	}
}
