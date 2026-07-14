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

-- name: UserProfileGetByUserID :one
SELECT
    *
FROM
    user_profiles
WHERE
    user_id = $1
LIMIT 1;

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

-- name: OutboxEventCreate :one
INSERT INTO outbox_events(event_type, payload)
    VALUES ($1, $2)
RETURNING
    *;

-- name: OutboxEventGetByID :one
SELECT
    *
FROM
    outbox_events
WHERE
    id = $1;

-- name: OutboxEventList :many
SELECT
    *
FROM
    outbox_events
WHERE ($1::uuid IS NULL
    OR id < $1)
ORDER BY
    id DESC
LIMIT $2;

-- name: OutboxEventAcquireBatch :many
UPDATE
    outbox_events
SET
    locked_by = $2,
    lease_expires_at = CURRENT_TIMESTAMP +($3::text || ' seconds')::interval
WHERE
    id IN (
        SELECT
            id
        FROM
            outbox_events
        WHERE
            processed_at IS NULL
            AND attempts < max_attempts
            AND next_attempt_at <= CURRENT_TIMESTAMP
            AND (lease_expires_at IS NULL
                OR lease_expires_at < CURRENT_TIMESTAMP)
        ORDER BY
            next_attempt_at ASC,
            id ASC
        LIMIT $1
        FOR UPDATE
            SKIP LOCKED)
RETURNING
    *;

-- name: OutboxEventMarkProcessed :one
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP,
    locked_by = NULL,
    lease_expires_at = NULL
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventRecordFailure :one
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    last_error = $2,
    locked_by = NULL,
    lease_expires_at = NULL,
    next_attempt_at = CURRENT_TIMESTAMP +(INTERVAL '1 minute' * POWER(2, attempts + 1)::int)
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventMarkDeadLetter :one
UPDATE
    outbox_events
SET
    last_error = COALESCE($2, 'Manually marked dead letter by operator.'),
    attempts = max_attempts,
    locked_by = NULL,
    lease_expires_at = NULL
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventResetAttempts :one
UPDATE
    outbox_events
SET
    attempts = 0,
    next_attempt_at = CURRENT_TIMESTAMP,
    last_error = NULL,
    locked_by = NULL,
    lease_expires_at = NULL
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventDeleteByID :exec
DELETE FROM outbox_events
WHERE id = $1;

-- name: OutboxEventPurgeProcessed :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - INTERVAL '7 days');

