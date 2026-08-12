package postgres

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type storedRatingState struct {
	ID        int64
	AvgTenths int
	Scores    map[string]int
}

func loadStoredRatingState(ctx context.Context, tx pgx.Tx, userID, titleID int64) (*storedRatingState, error) {
	rows, err := tx.Query(ctx, `
SELECT r.id, r.avg_score, c.code, rs.score
FROM ratings r
LEFT JOIN rating_scores rs ON rs.rating_id = r.id
LEFT JOIN criteria c ON c.id = rs.criterion_id
WHERE r.user_id = $1 AND r.title_id = $2
ORDER BY c.sort_order, c.id
FOR UPDATE OF r`, userID, titleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result *storedRatingState
	for rows.Next() {
		var (
			id    int64
			avg   pgtype.Numeric
			code  pgtype.Text
			score pgtype.Int2
		)
		if err := rows.Scan(&id, &avg, &code, &score); err != nil {
			return nil, err
		}
		if result == nil {
			result = &storedRatingState{ID: id, AvgTenths: int(math.Round(numericToFloat64(avg) * 10)), Scores: make(map[string]int)}
		}
		if code.Valid && score.Valid {
			result.Scores[code.String] = int(score.Int16)
		}
	}
	return result, rows.Err()
}

func insertRatingAchievementFact(
	ctx context.Context,
	tx pgx.Tx,
	kind string,
	actorID, titleID, ratingID int64,
	avgTenths int,
	scores map[string]int,
	previous *storedRatingState,
	occurredAt time.Time,
) error {
	scoresJSON, err := json.Marshal(scores)
	if err != nil {
		return err
	}
	previousScores := []byte(`{}`)
	var previousAvg *int
	if previous != nil {
		previousScores, err = json.Marshal(previous.Scores)
		if err != nil {
			return err
		}
		value := previous.AvgTenths
		previousAvg = &value
	}
	_, err = tx.Exec(ctx, `
INSERT INTO achievement_facts (
    kind, actor_id, title_id, entity_id, avg_tenths, previous_avg_tenths,
    scores, previous_scores, occurred_at
)
VALUES ($1::achievement_fact_kind, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)`,
		kind, actorID, titleID, ratingID, avgTenths, previousAvg, string(scoresJSON), string(previousScores), occurredAt)
	return err
}

func insertCommentAchievementFact(ctx context.Context, tx pgx.Tx, actorID, titleID, commentID, parentID int64, occurredAt time.Time) error {
	_, err := tx.Exec(ctx, `
INSERT INTO achievement_facts (kind, actor_id, title_id, entity_id, parent_entity_id, occurred_at)
VALUES ('comment_created', $1, $2, $3, NULLIF($4, 0), $5)
ON CONFLICT DO NOTHING`, actorID, titleID, commentID, parentID, occurredAt)
	return err
}

func insertWatchlistAchievementFact(ctx context.Context, tx pgx.Tx, actorID, titleID int64, occurredAt time.Time) error {
	_, err := tx.Exec(ctx, `
INSERT INTO achievement_facts (kind, actor_id, title_id, occurred_at)
VALUES ('watchlist_added', $1, $2, $3)`, actorID, titleID, occurredAt)
	return err
}

func equalScores(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for code, score := range left {
		if right[code] != score {
			return false
		}
	}
	return true
}
