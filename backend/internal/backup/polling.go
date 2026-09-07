package backup

import (
	"context"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
)

// UpdatesViaLongPolling retains the update offset across connection failures.
func UpdatesViaLongPolling(ctx context.Context, bot *telego.Bot, reconnect func(context.Context), logger *slog.Logger) <-chan telego.Update {
	updates := make(chan telego.Update, 100)
	go func() {
		defer close(updates)
		params := &telego.GetUpdatesParams{AllowedUpdates: []string{"message"}, Timeout: 30}
		for ctx.Err() == nil {
			pollCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
			batch, err := bot.GetUpdates(pollCtx, params)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Warn("telegram backup bot poll failed", "error", err)
				reconnect(ctx)
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
					continue
				}
			}
			for _, update := range batch {
				if update.UpdateID < params.Offset {
					continue
				}
				params.Offset = update.UpdateID + 1
				select {
				case <-ctx.Done():
					return
				case updates <- update.WithContext(ctx):
				}
			}
		}
	}()
	return updates
}
