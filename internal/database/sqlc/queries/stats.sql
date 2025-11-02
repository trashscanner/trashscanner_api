-- name: GetStatsByUserID :one
SELECT * FROM stats
WHERE user_id = $1;

-- name: UpdateStats :exec
UPDATE stats
SET
    status = $1,
    rating = $2,
    files_scanned = $3,
    total_weight = $4,
    achievements = $5,
    trash_by_types = $6,
    updated_at = now()
WHERE id = $7;