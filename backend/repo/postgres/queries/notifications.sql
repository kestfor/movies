-- name: CreateRatingActivityEvent :one
INSERT INTO activity_events (actor_id, title_id, kind, rating_id)
VALUES ($1, $2, 'rating_created', $3)
RETURNING id;

-- name: CreateCommentActivityEvent :one
INSERT INTO activity_events (actor_id, title_id, kind, comment_id)
VALUES ($1, $2, 'comment_created', $3)
RETURNING id;

-- name: DeliverActivityEventToFriends :exec
INSERT INTO notification_deliveries (user_id, event_id)
SELECT
    CASE
        WHEN f.requester_id = $1 THEN f.addressee_id
        ELSE f.requester_id
    END AS friend_id,
    $2
FROM friendships f
WHERE f.status = 'accepted'
  AND (f.requester_id = $1 OR f.addressee_id = $1)
ON CONFLICT DO NOTHING;

-- name: ListNotifications :many
SELECT
    ae.id AS event_id,
    ae.kind,
    ae.created_at,
    nd.read_at,
    actor.id AS actor_id,
    actor.uuid AS actor_uuid,
    actor.tg_id AS actor_tg_id,
    actor.username AS actor_username,
    actor.first_name AS actor_first_name,
    actor.photo_url AS actor_photo_url,
    actor.created_at AS actor_created_at,
    COALESCE(t.id, 0)::bigint AS title_id,
    COALESCE(t.tmdb_id, 0)::bigint AS tmdb_id,
    COALESCE(t.media_type, 'movie'::media_type) AS media_type,
    COALESCE(t.title, '')::text AS title,
    t.original_title,
    t.poster_path,
    t.release_year,
    COALESCE(t.genres, '[]'::jsonb) AS genres,
    t.overview,
    r.id AS rating_id,
    r.avg_score AS rating_avg_score,
    c.id AS comment_id,
    c.body AS comment_body,
    COALESCE(ua.id::text, '')::text AS achievement_award_id,
    COALESCE(ua.achievement_code, '')::text AS achievement_code
FROM notification_deliveries nd
JOIN activity_events ae ON ae.id = nd.event_id
JOIN users actor ON actor.id = ae.actor_id
LEFT JOIN titles t ON t.id = ae.title_id
LEFT JOIN ratings r ON r.id = ae.rating_id
LEFT JOIN comments c ON c.id = ae.comment_id
LEFT JOIN user_achievements ua ON ua.id = ae.achievement_id
WHERE nd.user_id = $1
  AND (
    $2::timestamptz IS NULL
    OR (ae.created_at, ae.id) < ($2::timestamptz, $3::bigint)
  )
ORDER BY ae.created_at DESC, ae.id DESC
LIMIT $4;

-- name: CountUnreadNotifications :one
SELECT count(*)::bigint
FROM notification_deliveries
WHERE user_id = $1
  AND read_at IS NULL;

-- name: MarkNotificationRead :execrows
UPDATE notification_deliveries
SET read_at = COALESCE(read_at, now())
WHERE user_id = $1
  AND event_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notification_deliveries
SET read_at = COALESCE(read_at, now())
WHERE user_id = $1
  AND read_at IS NULL;
