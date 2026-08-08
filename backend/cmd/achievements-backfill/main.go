package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"movies/backend/internal/config"
	postgresrepo "movies/backend/internal/repo/postgres"
	usecaseachievements "movies/backend/internal/usecase/achievements"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockKey int64 = 0x4b696e6f4b727567

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	lockConnection, err := pool.Acquire(ctx)
	if err != nil {
		logger.Error("acquire backfill connection", "error", err)
		os.Exit(1)
	}
	defer lockConnection.Release()
	if _, err := lockConnection.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		logger.Error("acquire backfill lock", "error", err)
		os.Exit(1)
	}
	defer lockConnection.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, advisoryLockKey)

	repository := postgresrepo.NewAchievementRepository(pool)
	service, err := usecaseachievements.NewService(repository, logger)
	if err != nil {
		logger.Error("initialize achievements", "error", err)
		os.Exit(1)
	}
	startedAt := time.Now()
	if _, err := service.EnsureCatalog(ctx, startedAt); err != nil {
		logger.Error("ensure achievement catalog", "error", err)
		os.Exit(1)
	}

	run, completed, err := prepareRun(ctx, lockConnection.Conn(), service.Fingerprint())
	if err != nil {
		logger.Error("prepare backfill run", "error", err)
		os.Exit(1)
	}
	if completed {
		logger.Info("achievement backfill already completed", "catalog_fingerprint", service.Fingerprint())
		return
	}

	batchSize := envInt("ACHIEVEMENT_BACKFILL_BATCH_SIZE", 100)
	for {
		userIDs, err := listUserIDs(ctx, pool, run.LastUserID, batchSize)
		if err != nil {
			failRun(context.Background(), pool, run.ID, err)
			logger.Error("list backfill users", "run_id", run.ID, "error", err)
			os.Exit(1)
		}
		if len(userIDs) == 0 {
			if _, err := pool.Exec(ctx, `
UPDATE achievement_backfill_runs
SET status = 'completed', completed_at = now(), error = NULL
WHERE id = $1`, run.ID); err != nil {
				logger.Error("complete backfill run", "run_id", run.ID, "error", err)
				os.Exit(1)
			}
			logger.Info("achievement backfill completed", "run_id", run.ID, "processed_users", run.ProcessedUsers, "awarded_count", run.AwardedCount)
			return
		}

		for _, userID := range userIDs {
			awards, err := service.EvaluateUser(ctx, userID, usecaseachievements.AwardSourceBackfill, time.Now())
			if err != nil {
				failRun(context.Background(), pool, run.ID, err)
				logger.Error("evaluate backfill user", "run_id", run.ID, "user_id", userID, "error", err)
				os.Exit(1)
			}
			run.LastUserID = userID
			run.ProcessedUsers++
			run.AwardedCount += int64(len(awards))
			if _, err := pool.Exec(ctx, `
UPDATE achievement_backfill_runs
SET last_user_id = $2, processed_users = $3, awarded_count = $4, error = NULL
WHERE id = $1`, run.ID, run.LastUserID, run.ProcessedUsers, run.AwardedCount); err != nil {
				failRun(context.Background(), pool, run.ID, err)
				logger.Error("checkpoint backfill run", "run_id", run.ID, "user_id", userID, "error", err)
				os.Exit(1)
			}
		}
	}
}

type backfillRun struct {
	ID             int64
	LastUserID     int64
	ProcessedUsers int64
	AwardedCount   int64
}

func prepareRun(ctx context.Context, connection *pgx.Conn, fingerprint string) (backfillRun, bool, error) {
	var completed bool
	if err := connection.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM achievement_backfill_runs
    WHERE catalog_fingerprint = $1 AND status = 'completed'
)`, fingerprint).Scan(&completed); err != nil {
		return backfillRun{}, false, err
	}
	if completed {
		return backfillRun{}, true, nil
	}

	var run backfillRun
	err := connection.QueryRow(ctx, `
SELECT id, COALESCE(last_user_id, 0), processed_users, awarded_count
FROM achievement_backfill_runs
WHERE catalog_fingerprint = $1 AND status IN ('running', 'failed')
ORDER BY started_at DESC
LIMIT 1`, fingerprint).Scan(&run.ID, &run.LastUserID, &run.ProcessedUsers, &run.AwardedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		err = connection.QueryRow(ctx, `
INSERT INTO achievement_backfill_runs (catalog_fingerprint, status)
VALUES ($1, 'running')
RETURNING id`, fingerprint).Scan(&run.ID)
		return run, false, err
	}
	if err != nil {
		return backfillRun{}, false, err
	}
	_, err = connection.Exec(ctx, `
UPDATE achievement_backfill_runs
SET status = 'running', completed_at = NULL, error = NULL
WHERE id = $1`, run.ID)
	return run, false, err
}

func listUserIDs(ctx context.Context, pool *pgxpool.Pool, afterID int64, limit int) ([]int64, error) {
	rows, err := pool.Query(ctx, `
SELECT id FROM users
WHERE id > $1
ORDER BY id
LIMIT $2`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func failRun(ctx context.Context, pool *pgxpool.Pool, runID int64, cause error) {
	_, _ = pool.Exec(ctx, `
UPDATE achievement_backfill_runs
SET status = 'failed', error = left($2, 4000)
WHERE id = $1`, runID, cause.Error())
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 || value > 1000 {
		return fallback
	}
	return value
}
