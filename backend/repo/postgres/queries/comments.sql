-- name: GetCommentForValidation :one
SELECT id, title_id, user_id, parent_id, body, is_deleted, created_at, updated_at
FROM comments
WHERE id = $1;

-- name: InsertComment :one
INSERT INTO comments (title_id, user_id, parent_id, body, updated_at)
VALUES ($1, $2, $3, $4, now())
RETURNING id, title_id, user_id, parent_id, body, is_deleted, created_at, updated_at;

-- name: ListCommentsByTitle :many
SELECT
    c.id,
    c.title_id,
    c.user_id,
    c.parent_id,
    c.body,
    c.is_deleted,
    c.created_at,
    c.updated_at,
    u.id AS author_id,
    u.tg_id AS author_tg_id,
    u.username AS author_username,
    u.first_name AS author_first_name,
    u.photo_url AS author_photo_url,
    u.created_at AS author_created_at
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.title_id = $1
ORDER BY c.created_at, c.id;

-- name: UpdateCommentBody :one
UPDATE comments
SET body = $2,
    updated_at = now()
WHERE id = $1 AND user_id = $3 AND is_deleted = false
RETURNING id, title_id, user_id, parent_id, body, is_deleted, created_at, updated_at;

-- name: SoftDeleteComment :one
UPDATE comments
SET body = '',
    is_deleted = true,
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, title_id, user_id, parent_id, body, is_deleted, created_at, updated_at;
