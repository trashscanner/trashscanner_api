-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING id;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 AND revoked = FALSE AND expires_at > now()
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = TRUE, revoked_at = now(), updated_at = now()
WHERE token_hash = $1;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens
SET revoked = TRUE, revoked_at = now(), updated_at = now()
WHERE user_id = $1 AND revoked = FALSE;

-- name: GetActiveTokensByUser :many
SELECT * FROM refresh_tokens
WHERE user_id = $1 AND revoked = FALSE AND expires_at > now()
ORDER BY created_at DESC;
