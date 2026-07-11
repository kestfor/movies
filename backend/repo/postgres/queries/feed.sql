-- name: ListFeedRatings :many
WITH visible_ratings AS (
    SELECT r.id
    FROM ratings r
    JOIN friendships f ON f.status = 'accepted'
        AND (
            (f.requester_id = $1 AND f.addressee_id = r.user_id)
            OR (f.addressee_id = $1 AND f.requester_id = r.user_id)
        )
    WHERE ($2::timestamptz IS NULL OR (r.created_at, r.id) < ($2::timestamptz, $3::bigint))
    ORDER BY r.created_at DESC, r.id DESC
    LIMIT $4
)
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
FROM visible_ratings vr
JOIN ratings r ON r.id = vr.id
JOIN users u ON u.id = r.user_id
JOIN titles t ON t.id = r.title_id
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
ORDER BY r.created_at DESC, r.id DESC, c.sort_order, c.id;
