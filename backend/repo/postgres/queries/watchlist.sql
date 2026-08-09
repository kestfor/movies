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

-- name: ListAcceptedFriendIDsByUUIDs :many
SELECT u.id
FROM friendships f
JOIN users u ON u.id = CASE
    WHEN f.requester_id = sqlc.arg(user_id) THEN f.addressee_id
    ELSE f.requester_id
END
WHERE (f.requester_id = sqlc.arg(user_id) OR f.addressee_id = sqlc.arg(user_id))
  AND f.status = 'accepted'
  AND u.uuid::text = ANY(sqlc.arg(friend_uuids)::text[])
ORDER BY u.id;

-- name: ListWatchlistMatches :many
WITH accepted_friends AS (
    SELECT CASE
        WHEN f.requester_id = sqlc.arg(user_id) THEN f.addressee_id
        ELSE f.requester_id
    END AS user_id
    FROM friendships f
    WHERE (f.requester_id = sqlc.arg(user_id) OR f.addressee_id = sqlc.arg(user_id))
      AND f.status = 'accepted'
),
circle_users AS (
    SELECT sqlc.arg(user_id)::bigint AS user_id
    UNION ALL
    SELECT user_id FROM accepted_friends
),
circle_watchers AS (
    SELECT w.user_id, w.title_id, w.created_at
    FROM watchlist_items w
    JOIN circle_users cu ON cu.user_id = w.user_id
),
candidate_titles AS (
    SELECT
        own.title_id,
        count(cw.user_id)::int AS matches_count,
        max(cw.created_at)::timestamptz AS latest_added_at
    FROM watchlist_items own
    JOIN circle_watchers cw ON cw.title_id = own.title_id
    WHERE own.user_id = sqlc.arg(user_id)
    GROUP BY own.title_id
    HAVING count(cw.user_id) >= 2
       AND (
           cardinality(sqlc.arg(friend_ids)::bigint[]) = 0
           OR count(*) FILTER (WHERE cw.user_id = ANY(sqlc.arg(friend_ids)::bigint[])) = cardinality(sqlc.arg(friend_ids)::bigint[])
       )
),
paged_titles AS (
    SELECT title_id, matches_count, latest_added_at
    FROM candidate_titles
    WHERE sqlc.narg(cursor_matches_count)::int IS NULL
       OR (matches_count, latest_added_at, title_id) < (
           sqlc.narg(cursor_matches_count)::int,
           sqlc.narg(cursor_latest_added_at)::timestamptz,
           sqlc.narg(cursor_title_id)::bigint
       )
    ORDER BY matches_count DESC, latest_added_at DESC, title_id DESC
    LIMIT sqlc.arg(page_limit)
)
SELECT
    p.matches_count,
    p.latest_added_at,
    t.id AS title_id,
    t.tmdb_id,
    t.media_type,
    t.title,
    t.original_title,
    t.poster_path,
    t.release_year,
    t.genres,
    t.overview,
    u.id AS watcher_id,
    u.uuid AS watcher_uuid,
    u.tg_id AS watcher_tg_id,
    u.username AS watcher_username,
    u.first_name AS watcher_first_name,
    u.photo_url AS watcher_photo_url,
    u.created_at AS watcher_created_at
FROM paged_titles p
JOIN titles t ON t.id = p.title_id
JOIN circle_watchers cw ON cw.title_id = p.title_id
JOIN users u ON u.id = cw.user_id
ORDER BY
    p.matches_count DESC,
    p.latest_added_at DESC,
    p.title_id DESC,
    CASE WHEN u.id = sqlc.arg(user_id) THEN 0 ELSE 1 END,
    lower(u.first_name),
    u.uuid;

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
