-- ==========================================
-- META
-- ==========================================
-- name: UserSessionCount :one
SELECT
    COUNT(*)
FROM
    user_sessions;

-- ==========================================
-- CREATE
-- ==========================================
-- name: UserSessionCreate :one
INSERT INTO user_sessions(id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: UserSessionListActiveByUserID :many
SELECT
    *
FROM
    user_sessions
WHERE
    user_id = $1
    AND is_blocked = FALSE
    AND expires_at > CURRENT_TIMESTAMP
ORDER BY
    last_seen_at DESC;

-- ==========================================
-- GET
-- ==========================================
-- name: UserSessionGetByID :one
SELECT
    *
FROM
    user_sessions
WHERE
    id = $1
LIMIT 1;

-- name: UserSessionGetByRefreshToken :one
SELECT
    *
FROM
    user_sessions
WHERE
    refresh_token = $1
    AND is_blocked = FALSE
    AND expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- ==========================================
-- UPDATE
-- ==========================================
-- name: UserSessionUpdateRefreshToken :one
UPDATE
    user_sessions
SET
    refresh_token = $2,
    expires_at = $3
WHERE
    id = $1
RETURNING
    *;

-- name: UserSessionUpdateLastSeen :one
UPDATE
    user_sessions
SET
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    id = $1
RETURNING
    *;

-- name: UserSessionMarkBlocked :one
UPDATE
    user_sessions
SET
    is_blocked = TRUE
WHERE
    id = $1
RETURNING
    *;

-- ==========================================
-- DELETE
-- ==========================================
-- name: UserSessionDelete :exec
DELETE FROM user_sessions
WHERE id = $1
    AND user_id = $2;

-- name: UserSessionDeleteAllExcept :exec
DELETE FROM user_sessions
WHERE user_id = $1
    AND id != $2;

-- name: UserSessionPurgeExpired :exec
DELETE FROM user_sessions
WHERE expires_at <= CURRENT_TIMESTAMP;

