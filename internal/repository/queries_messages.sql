-- ==========================================
-- CREATE
-- ==========================================
-- name: InsertMessage :one
INSERT INTO messages(channel_id, user_id, content)
    VALUES ($1, $2, $3)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: MessageListByChannel :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = $1
ORDER BY
    created_at DESC
LIMIT $2 OFFSET $3;

-- ==========================================
-- GET
-- ==========================================
-- name: MessageGetByID :one
SELECT
    *
FROM
    messages
WHERE
    id = $1
LIMIT 1;

-- ==========================================
-- DELETE
-- ==========================================
-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = $1;

