package backup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var ErrBackupAlreadyRunning = errors.New("backup already running")

type DumpRunner interface {
	Dump(ctx context.Context) (DumpResult, error)
}

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendDocument(ctx context.Context, chatID int64, path, filename, caption string) error
}

type ScheduleSettings interface {
	ScheduledEnabled(ctx context.Context) (bool, error)
	SetScheduledEnabled(ctx context.Context, enabled bool) error
}

type Service struct {
	dumper      DumpRunner
	sender      Sender
	settings    ScheduleSettings
	adminChatID map[int64]struct{}
	admins      []int64
	schedule    DailySchedule
	timezone    string
	logger      *slog.Logger
	mu          sync.Mutex
	running     bool
}

type ServiceConfig struct {
	AdminChatIDs []int64
	Schedule     DailySchedule
	TimezoneName string
}

func NewService(dumper DumpRunner, sender Sender, settings ScheduleSettings, cfg ServiceConfig, logger *slog.Logger) *Service {
	adminChatID := make(map[int64]struct{}, len(cfg.AdminChatIDs))
	for _, chatID := range cfg.AdminChatIDs {
		adminChatID[chatID] = struct{}{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		dumper:      dumper,
		sender:      sender,
		settings:    settings,
		adminChatID: adminChatID,
		admins:      append([]int64(nil), cfg.AdminChatIDs...),
		schedule:    cfg.Schedule,
		timezone:    cfg.TimezoneName,
		logger:      logger,
	}
}

func (s *Service) IsAdmin(chatID int64) bool {
	_, ok := s.adminChatID[chatID]
	return ok
}

func (s *Service) HandleCommand(ctx context.Context, chatID int64, text string) {
	if !s.IsAdmin(chatID) {
		s.logger.Warn("ignoring backup bot command from unauthorized chat", "chat_id", chatID)
		return
	}

	command := strings.Fields(text)
	if len(command) == 0 {
		return
	}

	switch strings.Split(command[0], "@")[0] {
	case "/backup":
		s.runManualBackup(ctx, chatID)
	case "/backup_schedule_status":
		s.sendScheduleStatus(ctx, chatID)
	case "/backup_schedule_on":
		s.setSchedule(ctx, chatID, true)
	case "/backup_schedule_off":
		s.setSchedule(ctx, chatID, false)
	}
}

func (s *Service) RunScheduledBackup(ctx context.Context) {
	enabled, err := s.settings.ScheduledEnabled(ctx)
	if err != nil {
		s.logger.Error("read scheduled backup setting", "error", err)
		s.notifyAdmins(ctx, "Не удалось прочитать статус расписания backup.")
		return
	}
	if !enabled {
		s.logger.Info("scheduled backup skipped because schedule is disabled")
		return
	}

	if err := s.runBackup(ctx, s.admins, "Ежедневный дамп базы данных"); err != nil {
		s.logger.Error("scheduled backup failed", "error", err)
		s.notifyAdmins(ctx, fmt.Sprintf("Ежедневный backup не выполнен: %s", err))
	}
}

func (s *Service) runManualBackup(ctx context.Context, chatID int64) {
	_ = s.sender.SendMessage(ctx, chatID, "Начинаю выгрузку дампа базы.")
	if err := s.runBackup(ctx, []int64{chatID}, "Дамп базы данных"); err != nil {
		s.logger.Error("manual backup failed", "chat_id", chatID, "error", err)
		_ = s.sender.SendMessage(ctx, chatID, fmt.Sprintf("Backup не выполнен: %s", err))
	}
}

func (s *Service) runBackup(ctx context.Context, chatIDs []int64, caption string) error {
	if !s.acquire() {
		return ErrBackupAlreadyRunning
	}
	defer s.release()

	dump, err := s.dumper.Dump(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(dump.Path); err != nil {
			s.logger.Warn("remove dump file", "path", dump.Path, "error", err)
		}
	}()

	for _, chatID := range chatIDs {
		if err := s.sender.SendDocument(ctx, chatID, dump.Path, dump.Name, caption); err != nil {
			return fmt.Errorf("send dump to chat %d: %w", chatID, err)
		}
	}
	return nil
}

func (s *Service) sendScheduleStatus(ctx context.Context, chatID int64) {
	enabled, err := s.settings.ScheduledEnabled(ctx)
	if err != nil {
		s.logger.Error("read scheduled backup setting", "chat_id", chatID, "error", err)
		_ = s.sender.SendMessage(ctx, chatID, "Не удалось прочитать статус расписания backup.")
		return
	}

	status := "выключено"
	if enabled {
		status = "включено"
	}
	_ = s.sender.SendMessage(ctx, chatID, fmt.Sprintf(
		"Расписание backup: %s\nВремя: %s\nTimezone: %s",
		status,
		s.schedule.String(),
		s.timezone,
	))
}

func (s *Service) setSchedule(ctx context.Context, chatID int64, enabled bool) {
	if err := s.settings.SetScheduledEnabled(ctx, enabled); err != nil {
		s.logger.Error("set scheduled backup setting", "chat_id", chatID, "enabled", enabled, "error", err)
		_ = s.sender.SendMessage(ctx, chatID, "Не удалось обновить расписание backup.")
		return
	}

	if enabled {
		_ = s.sender.SendMessage(ctx, chatID, "Ежедневная выгрузка backup включена.")
	} else {
		_ = s.sender.SendMessage(ctx, chatID, "Ежедневная выгрузка backup выключена.")
	}
}

func (s *Service) notifyAdmins(ctx context.Context, text string) {
	for _, chatID := range s.admins {
		if err := s.sender.SendMessage(ctx, chatID, text); err != nil {
			s.logger.Warn("notify backup admin", "chat_id", chatID, "error", err)
		}
	}
}

func (s *Service) acquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *Service) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
}
