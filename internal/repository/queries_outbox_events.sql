-- name: OutboxEventCreate :one
INSERT INTO outbox_events(created_at, updated_at, event_type, payload, processed_at, attempts, max_attempts, next_attempt_at, last_error)
    VALUES (COALESCE(sqlc.narg('created_at')::timestamptz, CURRENT_TIMESTAMP), COALESCE(sqlc.narg('updated_at')::timestamptz, CURRENT_TIMESTAMP), sqlc.arg('event_type')::varchar, sqlc.arg('payload')::jsonb, sqlc.narg('processed_at')::timestamptz, COALESCE(sqlc.narg('attempts')::int, 0), COALESCE(sqlc.narg('max_attempts')::int, 5), COALESCE(sqlc.narg('next_attempt_at')::timestamptz, CURRENT_TIMESTAMP), sqlc.narg('last_error')::text)
RETURNING
    *;

-- name: OutboxEventList :many
SELECT
    *
FROM
    outbox_events
ORDER BY
    created_at DESC
LIMIT $1 OFFSET $2;

-- name: OutboxEventCount :one
SELECT
    COUNT(*)
FROM
    outbox_events;

-- name: OutboxEventGetByID :one
SELECT
    *
FROM
    outbox_events
WHERE
    id = $1;

-- name: OutboxEventUpdateByID :one
UPDATE
    outbox_events
SET
    created_at = COALESCE($2, created_at),
    updated_at = COALESCE($3, updated_at),
    event_type = COALESCE($4, event_type),
    payload = COALESCE($5, payload),
    processed_at = COALESCE($6, processed_at),
    attempts = COALESCE($7, attempts),
    max_attempts = COALESCE($8, max_attempts),
    next_attempt_at = COALESCE($9, next_attempt_at),
    last_error = COALESCE($10, last_error)
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventDeleteByID :exec
DELETE FROM outbox_events
WHERE id = $1;

-- name: OutboxEventCountPending :one
SELECT
    COUNT(*)
FROM
    outbox_events
WHERE
    processed_at IS NULL;

-- name: OutboxEventMarkProcessed :exec
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: OutboxEventRecordFailure :exec
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    last_error = $2,
    -- Exponential backoff: 2^attempts minutes
    next_attempt_at = CURRENT_TIMESTAMP +(INTERVAL '1 minute' * POWER(2, attempts + 1))
WHERE
    id = $1;

-- name: OutboxEventResetAttempts :exec
UPDATE
    outbox_events
SET
    attempts = 0,
    next_attempt_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: OutboxEventMarkDeadLetter :exec
-- Permanently ignores an event that cannot be processed.
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP,
    last_error = $2,
    attempts = max_attempts
WHERE
    id = $1;

-- name: OutboxEventAcquireBatch :many
-- Uses a CTE to lock rows and immediately push next_attempt_at into the future.
-- This creates a "visibility timeout" so if the worker crashes, the events will naturally retry.
WITH batch AS (
    SELECT
        id
    FROM
        outbox_events
    WHERE
        processed_at IS NULL
        AND attempts < max_attempts
        AND next_attempt_at <= CURRENT_TIMESTAMP
    ORDER BY
        created_at ASC
    LIMIT $1
    FOR UPDATE
        SKIP LOCKED)
UPDATE
    outbox_events
SET
    next_attempt_at = CURRENT_TIMESTAMP + INTERVAL '5 minutes'
WHERE
    id IN (
        SELECT
            id
        FROM
            batch)
RETURNING
    *;

-- name: OutboxEventDeleteOld :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - INTERVAL '7 days');

