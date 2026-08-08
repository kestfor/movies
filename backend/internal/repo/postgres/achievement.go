package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	usecaseachievements "movies/backend/internal/usecase/achievements"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AchievementRepository struct {
	pool *pgxpool.Pool
}

func NewAchievementRepository(pool *pgxpool.Pool) *AchievementRepository {
	return &AchievementRepository{pool: pool}
}

func (r *AchievementRepository) EnsureCatalog(
	ctx context.Context,
	definitions []usecaseachievements.Definition,
	_ string,
	introducedAt time.Time,
) (map[string]time.Time, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for _, definition := range definitions {
		if _, err := tx.Exec(ctx, `
INSERT INTO achievement_catalog_state (achievement_code, definition_fingerprint, introduced_at)
VALUES ($1, $2, $3)
ON CONFLICT (achievement_code) DO NOTHING`, definition.Code, definitionFingerprint(definition), introducedAt); err != nil {
			return nil, err
		}
	}
	rows, err := tx.Query(ctx, `
SELECT achievement_code, introduced_at
FROM achievement_catalog_state
WHERE achievement_code = ANY($1::text[])`, definitionCodes(definitions))
	if err != nil {
		return nil, err
	}
	introduced := make(map[string]time.Time, len(definitions))
	for rows.Next() {
		var code string
		var at time.Time
		if err := rows.Scan(&code, &at); err != nil {
			rows.Close()
			return nil, err
		}
		introduced[code] = at
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(introduced) != len(definitions) {
		return nil, fmt.Errorf("achievement catalog state incomplete: got %d want %d", len(introduced), len(definitions))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return introduced, nil
}

func (r *AchievementRepository) LoadSnapshot(ctx context.Context, userID int64) (usecaseachievements.Snapshot, error) {
	snapshot := usecaseachievements.Snapshot{UserID: userID}
	history, current, err := r.loadRatingFacts(ctx, userID)
	if err != nil {
		return snapshot, err
	}
	snapshot.RatingHistory = history
	snapshot.Ratings = current
	friends, err := r.loadFriends(ctx, userID)
	if err != nil {
		return snapshot, err
	}
	snapshot.Friends = friends
	if err := r.attachFriendRatings(ctx, snapshot.Friends); err != nil {
		return snapshot, err
	}
	comments, err := r.loadComments(ctx)
	if err != nil {
		return snapshot, err
	}
	snapshot.Comments = comments
	watchlists, err := r.loadWatchlists(ctx, append(friendIDs(snapshot.Friends), userID))
	if err != nil {
		return snapshot, err
	}
	snapshot.Watchlist = watchlists[userID]
	for index := range snapshot.Friends {
		snapshot.Friends[index].Watchlist = watchlists[snapshot.Friends[index].User.ID]
	}
	return snapshot, nil
}

func (r *AchievementRepository) SaveEvaluation(ctx context.Context, params usecaseachievements.SaveEvaluationParams) ([]usecaseachievements.StoredAward, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for code, result := range params.Evaluation.Metrics {
		reachedAt := result.ReachedAt[result.Value]
		if reachedAt.IsZero() {
			reachedAt = params.EvaluatedAt
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO user_achievement_metrics (user_id, metric_code, value, reached_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, metric_code) DO UPDATE
SET value = GREATEST(user_achievement_metrics.value, EXCLUDED.value),
    reached_at = CASE
        WHEN EXCLUDED.value > user_achievement_metrics.value THEN EXCLUDED.reached_at
        ELSE user_achievement_metrics.reached_at
    END,
    updated_at = now()`, params.UserID, string(code), result.Value, reachedAt); err != nil {
			return nil, err
		}
	}
	inserted := make([]usecaseachievements.StoredAward, 0)
	for _, candidate := range params.Evaluation.Candidates {
		source := params.Source
		introducedAt := params.Introduced[candidate.Definition.Code]
		if source != usecaseachievements.AwardSourceBackfill && !introducedAt.IsZero() && !candidate.EarnedAt.After(introducedAt) {
			source = usecaseachievements.AwardSourceBackfill
		}
		var award usecaseachievements.StoredAward
		var awardedAt time.Time
		err := tx.QueryRow(ctx, `
INSERT INTO user_achievements (user_id, achievement_code, xp, earned_at, source)
VALUES ($1, $2, $3, $4, $5::achievement_award_source)
ON CONFLICT (user_id, achievement_code) DO NOTHING
RETURNING id::text, awarded_at`, params.UserID, candidate.Definition.Code, candidate.Definition.XP, candidate.EarnedAt, string(source)).Scan(&award.ID, &awardedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		award.UserID = params.UserID
		award.Code = candidate.Definition.Code
		award.XP = candidate.Definition.XP
		award.EarnedAt = candidate.EarnedAt
		award.AwardedAt = awardedAt
		award.Source = source
		inserted = append(inserted, award)
		if source != usecaseachievements.AwardSourceBackfill {
			if err := createAchievementActivity(ctx, tx, params.UserID, award.ID); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return inserted, nil
}

func (r *AchievementRepository) GetUserByUUID(ctx context.Context, rawUUID string) (domain.User, bool, error) {
	uuid, ok := uuidFromString(rawUUID)
	if !ok {
		return domain.User{}, false, nil
	}
	var (
		id                 int64
		tgID               int64
		username, photoURL pgtype.Text
		firstName          string
		createdAt          pgtype.Timestamptz
	)
	err := r.pool.QueryRow(ctx, `
SELECT id, tg_id, username, first_name, photo_url, created_at
FROM users WHERE uuid = $1`, uuid).Scan(&id, &tgID, &username, &firstName, &photoURL, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, false, nil
	}
	if err != nil {
		return domain.User{}, false, err
	}
	return domain.User{
		ID: id, UUID: rawUUID, TgID: tgID, Username: textToString(username),
		FirstName: firstName, PhotoURL: textToString(photoURL), CreatedAt: createdAt.Time,
	}, true, nil
}

func (r *AchievementRepository) GetRelationship(ctx context.Context, viewerID, userID int64) (string, error) {
	var relationship string
	err := r.pool.QueryRow(ctx, `
SELECT CASE
    WHEN $1::bigint = $2::bigint THEN 'self'
    WHEN f.status = 'accepted' THEN 'friend'
    WHEN f.status = 'pending' AND f.requester_id = $1 THEN 'outgoing'
    WHEN f.status = 'pending' AND f.addressee_id = $1 THEN 'incoming'
    ELSE 'none'
END
FROM users u
LEFT JOIN friendships f ON
    (f.requester_id = $1 AND f.addressee_id = u.id)
    OR (f.addressee_id = $1 AND f.requester_id = u.id)
WHERE u.id = $2`, viewerID, userID).Scan(&relationship)
	return relationship, err
}

func (r *AchievementRepository) ListCircleUserIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
SELECT CASE WHEN requester_id = $1 THEN addressee_id ELSE requester_id END
FROM friendships
WHERE status = 'accepted' AND (requester_id = $1 OR addressee_id = $1)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *AchievementRepository) ListAwards(ctx context.Context, userID int64) ([]usecaseachievements.StoredAward, error) {
	return r.listAwards(ctx, userID, false)
}

func (r *AchievementRepository) ListUnseen(ctx context.Context, userID int64) ([]usecaseachievements.StoredAward, error) {
	return r.listAwards(ctx, userID, true)
}

func (r *AchievementRepository) ListMetricValues(ctx context.Context, userID int64) (map[usecaseachievements.MetricCode]int64, error) {
	rows, err := r.pool.Query(ctx, `
SELECT metric_code, value
FROM user_achievement_metrics
WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[usecaseachievements.MetricCode]int64)
	for rows.Next() {
		var code string
		var value int64
		if err := rows.Scan(&code, &value); err != nil {
			return nil, err
		}
		result[usecaseachievements.MetricCode(code)] = value
	}
	return result, rows.Err()
}

func (r *AchievementRepository) ListLeaderboard(ctx context.Context, viewerID int64) ([]domain.LeaderboardEntry, error) {
	rows, err := r.pool.Query(ctx, `
WITH circle AS (
    SELECT $1::bigint AS user_id
    UNION
    SELECT CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
    FROM friendships f
    WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
), stats AS (
    SELECT c.user_id, COALESCE(sum(ua.xp), 0)::bigint AS total_xp, count(ua.id)::bigint AS unlocked_count
    FROM circle c
    LEFT JOIN user_achievements ua ON ua.user_id = c.user_id
    GROUP BY c.user_id
), ranked AS (
    SELECT s.*, dense_rank() OVER (ORDER BY s.total_xp DESC)::bigint AS rank
    FROM stats s
)
SELECT r.rank, u.id, u.uuid, u.tg_id, u.username, u.first_name, u.photo_url, u.created_at,
       r.total_xp, r.unlocked_count
FROM ranked r
JOIN users u ON u.id = r.user_id
ORDER BY r.total_xp DESC, r.unlocked_count DESC, lower(u.first_name), u.id`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.LeaderboardEntry, 0)
	for rows.Next() {
		var (
			rank, id, tgID, totalXP, unlocked int64
			uuid                              pgtype.UUID
			username, photoURL                pgtype.Text
			firstName                         string
			createdAt                         pgtype.Timestamptz
		)
		if err := rows.Scan(&rank, &id, &uuid, &tgID, &username, &firstName, &photoURL, &createdAt, &totalXP, &unlocked); err != nil {
			return nil, err
		}
		result = append(result, domain.LeaderboardEntry{
			Rank: int(rank), User: toDomainUserFields(id, uuid, tgID, username, firstName, photoURL, createdAt),
			TotalXP: int(totalXP), UnlockedCount: int(unlocked),
		})
	}
	return result, rows.Err()
}

func (r *AchievementRepository) MarkSeen(ctx context.Context, userID int64, awardIDs []string) error {
	_, err := r.pool.Exec(ctx, `
UPDATE user_achievements
SET seen_at = COALESCE(seen_at, now())
WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, awardIDs)
	return err
}

func (r *AchievementRepository) loadRatingFacts(ctx context.Context, userID int64) ([]usecaseachievements.RatingFact, []usecaseachievements.RatingFact, error) {
	rows, err := r.pool.Query(ctx, `
WITH known AS (
    SELECT user_id, title_id, created_at FROM ratings WHERE user_id = $1
    UNION ALL
    SELECT actor_id, title_id, created_at
    FROM activity_events
    WHERE actor_id = $1 AND kind = 'rating_created' AND title_id IS NOT NULL
), first_known AS (
    SELECT title_id, min(created_at) AS created_at
    FROM known GROUP BY title_id
)
SELECT fk.title_id, t.media_type, t.release_year, t.genres, fk.created_at,
       r.id, r.avg_score, r.updated_at
FROM first_known fk
JOIN titles t ON t.id = fk.title_id
LEFT JOIN ratings r ON r.user_id = $1 AND r.title_id = fk.title_id
ORDER BY fk.created_at, fk.title_id`, userID)
	if err != nil {
		return nil, nil, err
	}
	history := make([]usecaseachievements.RatingFact, 0)
	currentByTitle := make(map[int64]usecaseachievements.RatingFact)
	for rows.Next() {
		var (
			titleID     int64
			mediaType   gen.MediaType
			releaseYear pgtype.Int4
			genres      []byte
			createdAt   pgtype.Timestamptz
			ratingID    pgtype.Int8
			avgScore    pgtype.Numeric
			updatedAt   pgtype.Timestamptz
		)
		if err := rows.Scan(&titleID, &mediaType, &releaseYear, &genres, &createdAt, &ratingID, &avgScore, &updatedAt); err != nil {
			rows.Close()
			return nil, nil, err
		}
		fact := usecaseachievements.RatingFact{
			TitleID: titleID, MediaType: domain.MediaType(mediaType), ReleaseYear: int(releaseYear.Int32),
			Genres: unmarshalGenres(genres), CreatedAt: createdAt.Time,
		}
		history = append(history, fact)
		if ratingID.Valid {
			fact.AvgScore = numericToFloat64(avgScore)
			fact.UpdatedAt = updatedAt.Time
			fact.Scores = make(map[string]int)
			currentByTitle[titleID] = fact
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	if len(currentByTitle) > 0 {
		scoreRows, err := r.pool.Query(ctx, `
SELECT r.title_id, c.code, rs.score
FROM ratings r
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
WHERE r.user_id = $1`, userID)
		if err != nil {
			return nil, nil, err
		}
		for scoreRows.Next() {
			var titleID int64
			var code string
			var score int16
			if err := scoreRows.Scan(&titleID, &code, &score); err != nil {
				scoreRows.Close()
				return nil, nil, err
			}
			fact := currentByTitle[titleID]
			fact.Scores[code] = int(score)
			currentByTitle[titleID] = fact
		}
		if err := scoreRows.Err(); err != nil {
			scoreRows.Close()
			return nil, nil, err
		}
		scoreRows.Close()
	}
	current := make([]usecaseachievements.RatingFact, 0, len(currentByTitle))
	for _, fact := range currentByTitle {
		current = append(current, fact)
	}
	sort.Slice(current, func(i, j int) bool { return current[i].TitleID < current[j].TitleID })
	return history, current, nil
}

func (r *AchievementRepository) loadFriends(ctx context.Context, userID int64) ([]usecaseachievements.FriendFact, error) {
	rows, err := r.pool.Query(ctx, `
SELECT u.id, u.uuid, u.tg_id, u.username, u.first_name, u.photo_url, u.created_at, f.responded_at
FROM friendships f
JOIN users u ON u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
ORDER BY u.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]usecaseachievements.FriendFact, 0)
	for rows.Next() {
		var (
			id, tgID               int64
			uuid                   pgtype.UUID
			username, photoURL     pgtype.Text
			firstName              string
			createdAt, respondedAt pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &uuid, &tgID, &username, &firstName, &photoURL, &createdAt, &respondedAt); err != nil {
			return nil, err
		}
		result = append(result, usecaseachievements.FriendFact{
			User:       toDomainUserFields(id, uuid, tgID, username, firstName, photoURL, createdAt),
			AcceptedAt: respondedAt.Time,
		})
	}
	return result, rows.Err()
}

func (r *AchievementRepository) attachFriendRatings(ctx context.Context, friends []usecaseachievements.FriendFact) error {
	ids := friendIDs(friends)
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT r.user_id, r.title_id, t.media_type, t.release_year, t.genres,
       r.avg_score, r.created_at, r.updated_at, c.code, rs.score
FROM ratings r
JOIN titles t ON t.id = r.title_id
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
WHERE r.user_id = ANY($1::bigint[])
ORDER BY r.user_id, r.title_id, c.sort_order, c.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	type key struct{ userID, titleID int64 }
	byKey := make(map[key]*usecaseachievements.RatingFact)
	order := make([]key, 0)
	for rows.Next() {
		var (
			userID, titleID      int64
			mediaType            gen.MediaType
			releaseYear          pgtype.Int4
			genres               []byte
			avg                  pgtype.Numeric
			createdAt, updatedAt pgtype.Timestamptz
			code                 string
			score                int16
		)
		if err := rows.Scan(&userID, &titleID, &mediaType, &releaseYear, &genres, &avg, &createdAt, &updatedAt, &code, &score); err != nil {
			return err
		}
		itemKey := key{userID: userID, titleID: titleID}
		item := byKey[itemKey]
		if item == nil {
			item = &usecaseachievements.RatingFact{
				TitleID: titleID, MediaType: domain.MediaType(mediaType), ReleaseYear: int(releaseYear.Int32),
				Genres: unmarshalGenres(genres), AvgScore: numericToFloat64(avg), Scores: make(map[string]int),
				CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time,
			}
			byKey[itemKey] = item
			order = append(order, itemKey)
		}
		item.Scores[code] = int(score)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	byFriend := make(map[int64][]usecaseachievements.RatingFact)
	for _, itemKey := range order {
		byFriend[itemKey.userID] = append(byFriend[itemKey.userID], *byKey[itemKey])
	}
	for index := range friends {
		friends[index].Ratings = byFriend[friends[index].User.ID]
	}
	return nil
}

func (r *AchievementRepository) loadComments(ctx context.Context) ([]usecaseachievements.CommentFact, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, title_id, user_id, parent_id, created_at
FROM comments
ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]usecaseachievements.CommentFact, 0)
	for rows.Next() {
		var id, titleID, userID int64
		var parentID pgtype.Int8
		var createdAt pgtype.Timestamptz
		if err := rows.Scan(&id, &titleID, &userID, &parentID, &createdAt); err != nil {
			return nil, err
		}
		result = append(result, usecaseachievements.CommentFact{
			ID: id, TitleID: titleID, UserID: userID, ParentID: int8ToInt64(parentID), CreatedAt: createdAt.Time,
		})
	}
	return result, rows.Err()
}

func (r *AchievementRepository) loadWatchlists(ctx context.Context, userIDs []int64) (map[int64][]usecaseachievements.WatchlistFact, error) {
	result := make(map[int64][]usecaseachievements.WatchlistFact)
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT user_id, title_id, created_at
FROM watchlist_items
WHERE user_id = ANY($1::bigint[])
ORDER BY user_id, created_at, title_id`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, titleID int64
		var createdAt pgtype.Timestamptz
		if err := rows.Scan(&userID, &titleID, &createdAt); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], usecaseachievements.WatchlistFact{TitleID: titleID, CreatedAt: createdAt.Time})
	}
	return result, rows.Err()
}

func (r *AchievementRepository) listAwards(ctx context.Context, userID int64, unseenOnly bool) ([]usecaseachievements.StoredAward, error) {
	query := `
SELECT id::text, user_id, achievement_code, xp, earned_at, awarded_at, source::text, seen_at
FROM user_achievements
WHERE user_id = $1`
	if unseenOnly {
		query += ` AND seen_at IS NULL`
	}
	query += ` ORDER BY earned_at, achievement_code`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]usecaseachievements.StoredAward, 0)
	for rows.Next() {
		var award usecaseachievements.StoredAward
		var source string
		var seenAt pgtype.Timestamptz
		if err := rows.Scan(&award.ID, &award.UserID, &award.Code, &award.XP, &award.EarnedAt, &award.AwardedAt, &source, &seenAt); err != nil {
			return nil, err
		}
		award.Source = usecaseachievements.AwardSource(source)
		if seenAt.Valid {
			value := seenAt.Time
			award.SeenAt = &value
		}
		result = append(result, award)
	}
	return result, rows.Err()
}

func createAchievementActivity(ctx context.Context, tx pgx.Tx, userID int64, awardID string) error {
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO activity_events (actor_id, kind, achievement_id)
VALUES ($1, 'achievement_unlocked', $2::uuid)
RETURNING id`, userID, awardID).Scan(&eventID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
INSERT INTO notification_deliveries (user_id, event_id)
SELECT CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END, $2
FROM friendships f
WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
ON CONFLICT DO NOTHING`, userID, eventID)
	return err
}

func definitionCodes(definitions []usecaseachievements.Definition) []string {
	result := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, definition.Code)
	}
	return result
}

func definitionFingerprint(definition usecaseachievements.Definition) string {
	payload := fmt.Sprintf("%s|%s|%d|%d|%t", definition.Code, definition.Metric, definition.Target, definition.XP, definition.Secret)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func friendIDs(friends []usecaseachievements.FriendFact) []int64 {
	result := make([]int64, 0, len(friends))
	for _, friend := range friends {
		result = append(result, friend.User.ID)
	}
	return result
}
