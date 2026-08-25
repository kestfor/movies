package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"movies/backend/internal/backup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mymmrac/telego"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := backup.LoadConfig()
	if err != nil {
		logger.Error("load backup config", "error", err)
		os.Exit(1)
	}
	if !cfg.Enabled {
		logger.Info("backup bot disabled")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	settings := backup.NewSettingsStore(pool)
	if err := settings.EnsureScheduledEnabled(ctx, cfg.ScheduleEnabledDefault); err != nil {
		logger.Error("initialize backup settings", "error", err)
		os.Exit(1)
	}

	telegramConnection := backup.NewTelegramConnection(cfg.VLESSURL, logger)
	defer func() {
		if err := telegramConnection.Close(); err != nil {
			logger.Warn("close Telegram connection", "error", err)
		}
	}()

	bot, err := telego.NewBot(cfg.BotToken, telego.WithAPICaller(telegramConnection.Caller))
	if err != nil {
		logger.Error("create telegram bot", "error", err)
		os.Exit(1)
	}

	updates, err := bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		AllowedUpdates: []string{"message"},
		Timeout:        30,
	})
	if err != nil {
		logger.Error("start telegram long polling", "error", err)
		os.Exit(1)
	}

	svc := backup.NewService(
		&backup.Dumper{
			DatabaseURL: cfg.DatabaseURL,
			TmpDir:      cfg.TmpDir,
			Now: func() time.Time {
				return time.Now().In(cfg.Timezone)
			},
		},
		backup.NewTelegramSender(bot),
		settings,
		backup.ServiceConfig{
			AdminChatIDs: cfg.AdminChatIDs,
			Schedule:     cfg.Schedule,
			TimezoneName: cfg.TimezoneName,
		},
		logger,
	)

	scheduler := backup.Scheduler{
		Schedule: cfg.Schedule,
		Location: cfg.Timezone,
		Logger:   logger,
		Run:      svc.RunScheduledBackup,
	}
	go scheduler.Start(ctx)

	logger.Info("backup bot started", "schedule", cfg.Schedule.String(), "timezone", cfg.TimezoneName)
	for update := range updates {
		if update.Message == nil || update.Message.Text == "" {
			continue
		}
		svc.HandleCommand(ctx, update.Message.Chat.ID, update.Message.Text)
	}
	logger.Info("backup bot stopped")
}
