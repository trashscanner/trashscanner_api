-- name: CreateUser :one
INSERT INTO users (
    login,
    hashed_password
) VALUES (
    $1, $2
) RETURNING id;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted = FALSE;

-- name: GetUserByLogin :one
SELECT * FROM users
WHERE login = $1 AND deleted = FALSE;

-- name: UpdateUserPassword :exec
UPDATE users
SET hashed_password = $1, updated_at = now()
WHERE id = $2;

-- name: UpdateUserAvatar :exec
UPDATE users
SET avatar = $1, updated_at = now()
WHERE id = $2;

-- name: DeleteUser :exec
UPDATE users
SET deleted = TRUE, updated_at = now()
WHERE id = $1;
