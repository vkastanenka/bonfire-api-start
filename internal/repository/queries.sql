-- name: UserCreate :one
INSERT INTO users(id, email, username, password_hash)
    VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- name: UserGetByID :one
SELECT
    *
FROM
    users
WHERE
    id = $1
LIMIT 1;

-- name: UserGetByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1
LIMIT 1;

-- name: UserGetByUsername :one
SELECT
    *
FROM
    users
WHERE
    username = $1
LIMIT 1;

-- name: UserCheckAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            users u
        WHERE
            u.email = $1) AS email_available,
    NOT EXISTS (
        SELECT
            1
        FROM
            users u
        WHERE
            u.username = $2) AS username_available;

-- name: UserProfileCreate :one
INSERT INTO user_profiles(user_id, display_name)
    VALUES ($1, $2)
RETURNING
    *;

-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, refresh_token_hash, expires_at, client_ip, user_agent, os, browser)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING
    *;

-- name: SessionGetByID :one
SELECT
    *
FROM
    sessions
WHERE
    id = $1
LIMIT 1;

-- name: SessionUpdateRefreshToken :one
UPDATE
    sessions
SET
    refresh_token_hash = $2,
    expires_at = $3,
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    id = $1
    AND revoked_at IS NULL
    AND expires_at > CURRENT_TIMESTAMP
RETURNING
    *;

-- name: SessionUpdateLastSeen :one
UPDATE
    sessions
SET
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    id = $1
    AND revoked_at IS NULL
    AND expires_at > CURRENT_TIMESTAMP
RETURNING
    *;

-- name: SessionUpdateRevoked :one
UPDATE
    sessions
SET
    revoked_at = CURRENT_TIMESTAMP
WHERE
    id = $1
    AND revoked_at IS NULL
RETURNING
    *;

-- name: SessionDelete :exec
DELETE FROM sessions
WHERE id = $1;

-- name: SessionDeleteAllExcept :exec
DELETE FROM sessions
WHERE user_id = $1
    AND id != $2;

-- name: SessionDeleteAllExpired :exec
DELETE FROM sessions
WHERE expires_at <= CURRENT_TIMESTAMP;

