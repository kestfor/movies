-- name: GetFriendshipBetweenUsers :one
SELECT requester_id, addressee_id, status, created_at, responded_at
FROM friendships
WHERE (requester_id = $1 AND addressee_id = $2)
   OR (requester_id = $2 AND addressee_id = $1);

-- name: CreateFriendRequest :one
INSERT INTO friendships (requester_id, addressee_id, status)
VALUES ($1, $2, 'pending')
RETURNING requester_id, addressee_id, status, created_at, responded_at;

-- name: AcceptFriendRequest :one
UPDATE friendships
SET status = 'accepted',
    responded_at = now()
WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
RETURNING requester_id, addressee_id, status, created_at, responded_at;

-- name: DeletePendingFriendRequestBetweenUsers :execrows
DELETE FROM friendships
WHERE status = 'pending'
  AND (
      (requester_id = $1 AND addressee_id = $2)
      OR (requester_id = $2 AND addressee_id = $1)
  );

-- name: DeleteAcceptedFriendshipBetweenUsers :execrows
DELETE FROM friendships
WHERE status = 'accepted'
  AND (
      (requester_id = $1 AND addressee_id = $2)
      OR (requester_id = $2 AND addressee_id = $1)
  );

-- name: ListAcceptedFriends :many
SELECT
    u.id,
    u.tg_id,
    u.username,
    u.first_name,
    u.photo_url,
    u.created_at
FROM friendships f
JOIN users u ON u.id = CASE
    WHEN f.requester_id = $1 THEN f.addressee_id
    ELSE f.requester_id
END
WHERE (f.requester_id = $1 OR f.addressee_id = $1)
  AND f.status = 'accepted'
ORDER BY u.first_name, u.id;

-- name: ListIncomingFriendRequests :many
SELECT
    u.id,
    u.tg_id,
    u.username,
    u.first_name,
    u.photo_url,
    u.created_at,
    f.created_at AS requested_at
FROM friendships f
JOIN users u ON u.id = f.requester_id
WHERE f.addressee_id = $1
  AND f.status = 'pending'
ORDER BY f.created_at, f.requester_id;
