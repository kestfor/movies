package backup

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type fakeDumper struct {
	result DumpResult
	err    error
	calls  int
}

func (d *fakeDumper) Dump(context.Context) (DumpResult, error) {
	d.calls++
	if d.err != nil {
		return DumpResult{}, d.err
	}
	return d.result, nil
}

type fakeSender struct {
	messages  []string
	documents []string
}

func (s *fakeSender) SendMessage(_ context.Context, _ int64, text string) error {
	s.messages = append(s.messages, text)
	return nil
}

func (s *fakeSender) SendDocument(_ context.Context, _ int64, _, filename, caption string) error {
	s.documents = append(s.documents, filename+":"+caption)
	return nil
}

type fakeSettings struct {
	enabled bool
	err     error
}

func (s *fakeSettings) ScheduledEnabled(context.Context) (bool, error) {
	return s.enabled, s.err
}

func (s *fakeSettings) SetScheduledEnabled(_ context.Context, enabled bool) error {
	if s.err != nil {
		return s.err
	}
	s.enabled = enabled
	return nil
}

func TestHandleCommandBackup(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "movies.dump")
	dumper := &fakeDumper{result: DumpResult{Path: path, Name: "movies.dump"}}
	sender := &fakeSender{}
	service := NewService(dumper, sender, &fakeSettings{enabled: true}, ServiceConfig{
		AdminChatIDs: []int64{123},
		Schedule:     DailySchedule{Hour: 3},
		TimezoneName: "Asia/Krasnoyarsk",
	}, nil)

	service.HandleCommand(context.Background(), 123, "/backup")

	if dumper.calls != 1 {
		t.Fatalf("dump calls: got %d, want 1", dumper.calls)
	}
	if len(sender.documents) != 1 || sender.documents[0] != "movies.dump:Дамп базы данных" {
		t.Fatalf("unexpected documents: %+v", sender.documents)
	}
}

func TestHandleCommandRejectsUnauthorizedChat(t *testing.T) {
	t.Parallel()

	dumper := &fakeDumper{}
	sender := &fakeSender{}
	service := NewService(dumper, sender, &fakeSettings{enabled: true}, ServiceConfig{
		AdminChatIDs: []int64{123},
	}, nil)

	service.HandleCommand(context.Background(), 999, "/backup")

	if dumper.calls != 0 {
		t.Fatalf("dump calls: got %d, want 0", dumper.calls)
	}
	if len(sender.messages) != 0 || len(sender.documents) != 0 {
		t.Fatalf("unexpected sender calls: messages=%+v documents=%+v", sender.messages, sender.documents)
	}
}

func TestHandleCommandScheduleToggleAndStatus(t *testing.T) {
	t.Parallel()

	settings := &fakeSettings{enabled: false}
	sender := &fakeSender{}
	service := NewService(&fakeDumper{}, sender, settings, ServiceConfig{
		AdminChatIDs: []int64{123},
		Schedule:     DailySchedule{Hour: 3, Minute: 30},
		TimezoneName: "Asia/Krasnoyarsk",
	}, nil)

	service.HandleCommand(context.Background(), 123, "/backup_schedule_on")
	if !settings.enabled {
		t.Fatal("expected schedule to be enabled")
	}

	service.HandleCommand(context.Background(), 123, "/backup_schedule_status")
	if len(sender.messages) == 0 || !strings.Contains(sender.messages[len(sender.messages)-1], "включено") {
		t.Fatalf("unexpected status messages: %+v", sender.messages)
	}

	service.HandleCommand(context.Background(), 123, "/backup_schedule_off")
	if settings.enabled {
		t.Fatal("expected schedule to be disabled")
	}
}

func TestRunScheduledBackupSkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	dumper := &fakeDumper{}
	service := NewService(dumper, &fakeSender{}, &fakeSettings{enabled: false}, ServiceConfig{
		AdminChatIDs: []int64{123},
	}, nil)

	service.RunScheduledBackup(context.Background())

	if dumper.calls != 0 {
		t.Fatalf("dump calls: got %d, want 0", dumper.calls)
	}
}

func TestRunScheduledBackupNotifiesOnSettingsError(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(&fakeDumper{}, sender, &fakeSettings{err: errors.New("db down")}, ServiceConfig{
		AdminChatIDs: []int64{123},
	}, nil)

	service.RunScheduledBackup(context.Background())

	if len(sender.messages) != 1 {
		t.Fatalf("messages: got %d, want 1", len(sender.messages))
	}
}
