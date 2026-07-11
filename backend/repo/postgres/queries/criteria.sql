-- name: ListActiveCriteria :many
SELECT id, code, name, sort_order, is_active
FROM criteria
WHERE is_active = true
ORDER BY sort_order, id;
