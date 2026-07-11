-- name: UpsertUserByTelegramID :one
INSERT INTO users (tg_id, username, first_name, photo_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tg_id) DO UPDATE
SET username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    photo_url = EXCLUDED.photo_url
RETURNING id, tg_id, username, first_name, photo_url, created_at;

-- name: GetUserByID :one
SELECT id, tg_id, username, first_name, photo_url, created_at
FROM users
WHERE id = $1;
