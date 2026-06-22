-- ==========================================
-- META
-- ==========================================
-- name: UserDeleteRequestCount :one
SELECT
    COUNT(*)
FROM
    user_delete_requests;

-- ==========================================
-- CREATE
-- ==========================================
-- name: UserDeleteRequestCreate :one
INSERT INTO user_delete_requests(user_id, scheduled_at)
    VALUES ($1, $2)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: UserDeleteRequestListDue :many
SELECT
    user_id
FROM
    user_delete_requests
WHERE
    scheduled_at <= CURRENT_TIMESTAMP
ORDER BY
    scheduled_at ASC;

-- ==========================================
-- GET
-- ==========================================
-- name: UserDeleteRequestGetByUserID :one
SELECT
    *
FROM
    user_delete_requests
WHERE
    user_id = $1
LIMIT 1;

-- ==========================================
-- DELETE
-- ==========================================
-- name: UserDeleteRequestDeleteByUserID :exec
DELETE FROM user_delete_requests
WHERE user_id = $1;

