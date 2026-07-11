-- name: UpsertUserByTelegramID :one
INSERT INTO users (tg_id, username, first_name, photo_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tg_id) DO UPDATE
SET username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    photo_url = EXCLUDED.photo_url
RETURNING id, uuid, tg_id, username, first_name, photo_url, created_at;

-- name: GetUserByID :one
SELECT id, uuid, tg_id, username, first_name, photo_url, created_at
FROM users
WHERE id = $1;

-- name: GetUserByUUID :one
SELECT id, uuid, tg_id, username, first_name, photo_url, created_at
FROM users
WHERE uuid = $1;

-- name: SearchUsersByUsernamePrefix :many
SELECT
    u.id,
    u.uuid,
    u.tg_id,
    u.username,
    u.first_name,
    u.photo_url,
    u.created_at,
    CASE
        WHEN u.id = $1 THEN 'self'
        WHEN f.status = 'accepted' THEN 'friend'
        WHEN f.status = 'pending' AND f.requester_id = $1 THEN 'outgoing'
        WHEN f.status = 'pending' AND f.addressee_id = $1 THEN 'incoming'
        ELSE 'none'
    END AS relationship
FROM users u
LEFT JOIN friendships f ON
    (f.requester_id = $1 AND f.addressee_id = u.id)
    OR (f.addressee_id = $1 AND f.requester_id = u.id)
WHERE u.username IS NOT NULL
  AND lower(u.username) LIKE lower($2) || '%'
ORDER BY
    CASE WHEN u.id = $1 THEN 1 ELSE 0 END,
    u.username,
    u.id
LIMIT $3;

-- name: GetUserRelationship :one
SELECT
    CASE
        WHEN $1::bigint = $2::bigint THEN 'self'
        WHEN f.status = 'accepted' THEN 'friend'
        WHEN f.status = 'pending' AND f.requester_id = $1 THEN 'outgoing'
        WHEN f.status = 'pending' AND f.addressee_id = $1 THEN 'incoming'
        ELSE 'none'
    END AS relationship
FROM users u
LEFT JOIN friendships f ON
    (f.requester_id = $1 AND f.addressee_id = u.id)
    OR (f.addressee_id = $1 AND f.requester_id = u.id)
WHERE u.id = $2;
