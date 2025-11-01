-- name: CreateNewPrediction :one
INSERT INTO predictions (
    user_id,
    trash_scan,
    status
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: CompletePrediction :exec
UPDATE predictions
SET status = $1, result = $2, error = $3, updated_at = now()
WHERE id = $4;

-- name: GetPrediction :one
SELECT * FROM predictions
WHERE id = $1;

-- name: GetPredictionsByUserID :many
SELECT * FROM predictions
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;
