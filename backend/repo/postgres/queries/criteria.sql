-- name: ListActiveCriteria :many
SELECT id, code, name, description, sort_order, is_active
FROM criteria
WHERE is_active = true
ORDER BY sort_order, id;
