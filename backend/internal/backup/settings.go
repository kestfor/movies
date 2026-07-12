package backup

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

const scheduledEnabledKey = "scheduled_enabled"

type SettingsStore struct {
	pool *pgxpool.Pool
}

func NewSettingsStore(pool *pgxpool.Pool) *SettingsStore {
	return &SettingsStore{pool: pool}
}

func (s *SettingsStore) EnsureScheduledEnabled(ctx context.Context, defaultValue bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO backup_settings (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO NOTHING
	`, scheduledEnabledKey, strconv.FormatBool(defaultValue))
	if err != nil {
		return fmt.Errorf("ensure scheduled backup setting: %w", err)
	}
	return nil
}

func (s *SettingsStore) ScheduledEnabled(ctx context.Context) (bool, error) {
	var value string
	if err := s.pool.QueryRow(ctx, `
		SELECT value
		FROM backup_settings
		WHERE key = $1
	`, scheduledEnabledKey).Scan(&value); err != nil {
		return false, fmt.Errorf("read scheduled backup setting: %w", err)
	}

	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse scheduled backup setting: %w", err)
	}
	return enabled, nil
}

func (s *SettingsStore) SetScheduledEnabled(ctx context.Context, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO backup_settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    updated_at = now()
	`, scheduledEnabledKey, strconv.FormatBool(enabled))
	if err != nil {
		return fmt.Errorf("set scheduled backup setting: %w", err)
	}
	return nil
}
