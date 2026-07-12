-- name: ListCriteriaByCodes :many
SELECT id, code, name, sort_order, is_active
FROM criteria
WHERE is_active = true AND code = ANY($1::text[]);

-- name: UpsertRating :one
INSERT INTO ratings (user_id, title_id, avg_score, updated_at)
VALUES ($1, $2, $3::numeric, now())
ON CONFLICT (user_id, title_id) DO UPDATE
SET avg_score = EXCLUDED.avg_score,
    updated_at = now()
RETURNING id, user_id, title_id, avg_score, created_at, updated_at, (xmax = 0)::boolean AS inserted;

-- name: DeleteRatingScores :exec
DELETE FROM rating_scores
WHERE rating_id = $1;

-- name: InsertRatingScore :exec
INSERT INTO rating_scores (rating_id, criterion_id, score)
VALUES ($1, $2, $3);

-- name: DeleteRatingForUserTitle :exec
DELETE FROM ratings
WHERE user_id = $1 AND title_id = $2;

-- name: GetRatingByUserTitle :many
SELECT
    r.id,
    r.user_id,
    r.title_id,
    r.avg_score,
    r.created_at,
    r.updated_at,
    u.id AS author_id,
    u.uuid AS author_uuid,
    u.tg_id AS author_tg_id,
    u.username AS author_username,
    u.first_name AS author_first_name,
    u.photo_url AS author_photo_url,
    u.created_at AS author_created_at,
    c.code AS criterion_code,
    rs.score AS criterion_score
FROM ratings r
JOIN users u ON u.id = r.user_id
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
WHERE r.user_id = $1 AND r.title_id = $2
ORDER BY c.sort_order, c.id;

-- name: ListFriendRatingsByTitle :many
SELECT
    r.id,
    r.user_id,
    r.title_id,
    r.avg_score,
    r.created_at,
    r.updated_at,
    u.id AS author_id,
    u.uuid AS author_uuid,
    u.tg_id AS author_tg_id,
    u.username AS author_username,
    u.first_name AS author_first_name,
    u.photo_url AS author_photo_url,
    u.created_at AS author_created_at,
    c.code AS criterion_code,
    rs.score AS criterion_score
FROM ratings r
JOIN users u ON u.id = r.user_id
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
JOIN friendships f ON f.status = 'accepted'
    AND (
        (f.requester_id = $1 AND f.addressee_id = r.user_id)
        OR (f.addressee_id = $1 AND f.requester_id = r.user_id)
    )
WHERE r.title_id = $2
ORDER BY r.created_at DESC, r.id DESC, c.sort_order, c.id;

-- name: CountCommentsByTitle :one
SELECT count(*)::bigint
FROM comments
WHERE title_id = $1;

-- name: UserCanSeeRatings :one
SELECT EXISTS (
    SELECT 1
    WHERE $1::bigint = $2::bigint
    UNION ALL
    SELECT 1
    FROM friendships
    WHERE status = 'accepted'
      AND (
          (requester_id = $1 AND addressee_id = $2)
          OR (requester_id = $2 AND addressee_id = $1)
      )
);

-- name: ListUserRatings :many
SELECT
    r.id,
    r.user_id,
    r.title_id,
    r.avg_score,
    r.created_at,
    r.updated_at,
    t.id AS title_id,
    t.tmdb_id,
    t.media_type,
    t.title,
    t.original_title,
    t.poster_path,
    t.release_year,
    t.genres,
    t.overview,
    c.code AS criterion_code,
    rs.score AS criterion_score
FROM ratings r
JOIN titles t ON t.id = r.title_id
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
WHERE r.user_id = $1
ORDER BY r.created_at DESC, r.id DESC, c.sort_order, c.id;
