-- ==========================================
-- META
-- ==========================================
-- name: DeleteRequestCount :one
SELECT
    COUNT(*)
FROM
    delete_requests;

-- ==========================================
-- CREATE
-- ==========================================
-- name: DeleteRequestCreate :one
INSERT INTO delete_requests(user_id, scheduled_at)
    VALUES ($1, $2)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: DeleteRequestListDue :many
SELECT
    *
FROM
    delete_requests
WHERE
    scheduled_at <= CURRENT_TIMESTAMP
ORDER BY
    scheduled_at ASC;

-- ==========================================
-- GET
-- ==========================================
-- name: DeleteRequestGetByUserID :one
SELECT
    *
FROM
    delete_requests
WHERE
    user_id = $1
LIMIT 1;

-- ==========================================
-- DELETE
-- ==========================================
-- name: DeleteRequestDeleteByUserID :exec
DELETE FROM delete_requests
WHERE user_id = $1;

