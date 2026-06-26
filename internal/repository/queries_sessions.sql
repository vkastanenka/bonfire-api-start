-- ==========================================
-- META
-- ==========================================
-- name: SessionCount :one
SELECT
    COUNT(*)
FROM
    sessions;

-- ==========================================
-- CREATE
-- ==========================================
-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: SessionListByUserID :many
SELECT
    *
FROM
    sessions
WHERE
    user_id = $1
ORDER BY
    last_seen_at DESC;

-- name: SessionListActiveByUserID :many
SELECT
    *
FROM
    sessions
WHERE
    user_id = $1
    AND is_blocked = FALSE
    AND expires_at > CURRENT_TIMESTAMP
ORDER BY
    last_seen_at DESC;

-- name: SessionListExpiredByUserID :many
SELECT
    *
FROM
    sessions
WHERE
    user_id = $1
    AND expires_at <= CURRENT_TIMESTAMP
ORDER BY
    last_seen_at DESC;

-- name: SessionListBlockedByUserID :many
SELECT
    *
FROM
    sessions
WHERE
    user_id = $1
    AND is_blocked = TRUE
ORDER BY
    last_seen_at DESC;

-- ==========================================
-- GET
-- ==========================================
-- name: SessionGetByID :one
SELECT
    *
FROM
    sessions
WHERE
    id = $1
LIMIT 1;

-- name: SessionGetByRefreshToken :one
SELECT
    *
FROM
    sessions
WHERE
    refresh_token = $1
    AND is_blocked = FALSE
    AND expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- ==========================================
-- UPDATE
-- ==========================================
-- name: SessionUpdateRefreshToken :one
UPDATE
    sessions
SET
    refresh_token = $2,
    expires_at = $3
WHERE
    id = $1
RETURNING
    *;

-- name: SessionUpdateLastSeen :one
UPDATE
    sessions
SET
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    id = $1
RETURNING
    *;

-- name: SessionMarkBlocked :one
UPDATE
    sessions
SET
    is_blocked = TRUE
WHERE
    id = $1
RETURNING
    *;

-- ==========================================
-- DELETE
-- ==========================================
-- name: SessionDelete :exec
DELETE FROM sessions
WHERE id = $1
    AND user_id = $2;

-- name: SessionDeleteAllExcept :exec
DELETE FROM sessions
WHERE user_id = $1
    AND id != $2;

-- name: SessionPurgeExpired :exec
DELETE FROM sessions
WHERE expires_at <= CURRENT_TIMESTAMP;

