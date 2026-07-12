package backup

import (
	"context"
	"log/slog"
	"time"
)

type Scheduler struct {
	Schedule DailySchedule
	Location *time.Location
	Logger   *slog.Logger
	Run      func(context.Context)
}

func (s Scheduler) Start(ctx context.Context) {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}

	for {
		next := nextRun(time.Now().In(s.Location), s.Schedule, s.Location)
		timer := time.NewTimer(time.Until(next))
		logger.Info("scheduled next backup", "at", next.Format(time.RFC3339))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if s.Run != nil {
				s.Run(ctx)
			}
		}
	}
}

func nextRun(now time.Time, schedule DailySchedule, location *time.Location) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), schedule.Hour, schedule.Minute, 0, 0, location)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
