-- name: CreateUser :one
WITH new_user AS (
    INSERT INTO users (
        name,
        login,
        hashed_password,
        role
    ) VALUES (
        $1, $2, $3, $4
    ) RETURNING id
),
new_stats AS (
    INSERT INTO stats (user_id)
    SELECT id FROM new_user
    RETURNING id
)
SELECT new_user.id as user_id
FROM new_user;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted = FALSE;

-- name: GetUserByLogin :one
SELECT * FROM users
WHERE login = $1 AND deleted = FALSE;

-- name: UpdateUser :exec
UPDATE users
SET name = $1, updated_at = now()
WHERE id = $2;

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
