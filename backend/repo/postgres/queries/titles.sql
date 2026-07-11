-- name: GetTitleIDByTMDB :one
SELECT id
FROM titles
WHERE tmdb_id = $1 AND media_type = $2;

-- name: UpsertTitleSnapshot :one
INSERT INTO titles (
    tmdb_id,
    media_type,
    title,
    original_title,
    poster_path,
    release_year,
    genres,
    overview,
    cached_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, now())
ON CONFLICT (tmdb_id, media_type) DO UPDATE
SET title = EXCLUDED.title,
    original_title = EXCLUDED.original_title,
    poster_path = EXCLUDED.poster_path,
    release_year = EXCLUDED.release_year,
    genres = EXCLUDED.genres,
    overview = EXCLUDED.overview,
    cached_at = now()
RETURNING id;
