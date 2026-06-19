-- name: UserSessionCreate :one
INSERT INTO user_sessions(id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING
    *;

-- name: UserSessionGet :one
SELECT
    *
FROM
    user_sessions
WHERE
    refresh_token = $1
LIMIT 1;

-- name: UserSessionGetByID :one
SELECT
    *
FROM
    user_sessions
WHERE
    id = $1
LIMIT 1;


-- name: UserSessionListByUser :many
SELECT
    *
FROM
    user_sessions
WHERE
    user_id = $1
    AND is_blocked = FALSE
ORDER BY
    last_seen_at DESC;

-- name: UserSessionUpdateRefreshToken :exec
UPDATE
    user_sessions
SET
    refresh_token = $2,
    expires_at = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: UserSessionUpdateLastSeen :exec
UPDATE
    user_sessions
SET
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: UserSessionMarkBlocked :exec
UPDATE
    user_sessions
SET
    is_blocked = TRUE
WHERE
    id = $1;

-- name: UserSessionDelete :exec
DELETE FROM user_sessions
WHERE id = $1
    AND user_id = $2;

-- name: UserSessionDeleteAllExcept :exec
DELETE FROM user_sessions
WHERE user_id = $1
    AND id != $2;

