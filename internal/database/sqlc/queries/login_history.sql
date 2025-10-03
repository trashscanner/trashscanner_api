-- name: CreateLoginHistory :one
INSERT INTO login_history (
    user_id,
    login_attempt,
    success,
    failure_reason,
    ip_address,
    user_agent,
    location
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING id;

-- name: GetLoginHistoryByUser :many
SELECT * FROM login_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
