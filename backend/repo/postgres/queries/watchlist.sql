-- name: HasUserRatingForTitle :one
SELECT EXISTS (
    SELECT 1
    FROM ratings
    WHERE user_id = $1 AND title_id = $2
);

-- name: LockTitleForUpdate :one
SELECT id
FROM titles
WHERE id = $1
FOR UPDATE;

-- name: AddWatchlistItem :exec
INSERT INTO watchlist_items (user_id, title_id)
VALUES ($1, $2)
ON CONFLICT (user_id, title_id) DO NOTHING;

-- name: DeleteWatchlistItem :exec
DELETE FROM watchlist_items
WHERE user_id = $1 AND title_id = $2;

-- name: IsInWatchlist :one
SELECT EXISTS (
    SELECT 1
    FROM watchlist_items
    WHERE user_id = $1 AND title_id = $2
);

-- name: ListWatchlistTitleRefs :many
SELECT t.tmdb_id, t.media_type
FROM watchlist_items w
JOIN titles t ON t.id = w.title_id
WHERE w.user_id = $1;

-- name: ListWatchlistItems :many
WITH page_items AS (
    SELECT user_id, title_id, created_at
    FROM watchlist_items
    WHERE user_id = $1
      AND ($2::timestamptz IS NULL OR (created_at, title_id) < ($2::timestamptz, $3::bigint))
    ORDER BY created_at DESC, title_id DESC
    LIMIT $4
)
SELECT
    p.created_at AS added_at,
    t.id,
    t.tmdb_id,
    t.media_type,
    t.title,
    t.original_title,
    t.poster_path,
    t.release_year,
    t.genres,
    t.overview
FROM page_items p
JOIN titles t ON t.id = p.title_id
ORDER BY p.created_at DESC, p.title_id DESC;

-- name: CountWatchlistItems :one
SELECT count(*)::bigint
FROM watchlist_items
WHERE user_id = $1;

-- name: ListRatedTitleRefs :many
SELECT t.tmdb_id, t.media_type
FROM ratings r
JOIN titles t ON t.id = r.title_id
WHERE r.user_id = $1
ORDER BY r.id;

-- name: ListRecommendationSeeds :many
SELECT
    r.id AS rating_id,
    r.avg_score,
    r.updated_at,
    t.tmdb_id,
    t.media_type,
    t.title,
    t.genres
FROM ratings r
JOIN titles t ON t.id = r.title_id
WHERE r.user_id = $1 AND r.avg_score >= 7.0
ORDER BY r.avg_score DESC, r.updated_at DESC, r.id DESC
LIMIT 5;

-- name: ListUserRatingGenres :many
SELECT r.avg_score, t.genres
FROM ratings r
JOIN titles t ON t.id = r.title_id
WHERE r.user_id = $1
ORDER BY r.id;

-- name: CountUserRatings :one
SELECT count(*)::bigint
FROM ratings
WHERE user_id = $1;

-- name: GetUserRatingStats :one
SELECT
    count(*)::bigint AS count,
    COALESCE(avg(avg_score), 0)::numeric AS avg_score
FROM ratings
WHERE user_id = $1;
