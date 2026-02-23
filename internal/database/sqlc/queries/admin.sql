-- name: GetAdminUsers :many
SELECT 
    u.id, 
    u.login, 
    u.name, 
    u.role, 
    u.avatar, 
    u.deleted, 
    u.created_at, 
    u.updated_at,
    s.status,
    s.rating,
    s.files_scanned,
    s.total_weight,
    s.last_scanned_at,
    lh.last_login_at
FROM users u
LEFT JOIN stats s ON s.user_id = u.id
LEFT JOIN (
    SELECT user_id, MAX(created_at) AS last_login_at
    FROM login_history
    WHERE success = true
    GROUP BY user_id
) lh ON lh.user_id = u.id
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAdminUserByID :one
SELECT 
    u.id, 
    u.login, 
    u.name, 
    u.role, 
    u.avatar, 
    u.deleted, 
    u.created_at, 
    u.updated_at,
    s.status,
    s.rating,
    s.files_scanned,
    s.total_weight,
    s.last_scanned_at,
    lh.last_login_at
FROM users u
LEFT JOIN stats s ON s.user_id = u.id
LEFT JOIN (
    SELECT user_id, MAX(created_at) AS last_login_at
    FROM login_history
    WHERE success = true
    GROUP BY user_id
) lh ON lh.user_id = u.id
WHERE u.id = $1;

-- name: CountUsers :one
SELECT COUNT(id) FROM users;
