-- ==========================================
-- META
-- ==========================================
-- name: OutboxEventCount :one
SELECT
    COUNT(*)
FROM
    outbox_events;

-- ==========================================
-- CREATE
-- ==========================================
-- name: OutboxEventCreate :one
INSERT INTO outbox_events(event_type, payload)
    VALUES ($1, $2)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: OutboxEventList :many
SELECT
    id,
    created_at,
    updated_at,
    event_type,
    payload,
    processed_at,
    attempts,
    max_attempts,
    next_attempt_at,
    last_error
FROM
    outbox_events
WHERE ($1::uuid IS NULL
    OR id < $1)
ORDER BY
    id DESC
LIMIT $2;

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
        id ASC
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

-- ==========================================
-- GET
-- ==========================================
-- name: OutboxEventGetByID :one
SELECT
    id,
    created_at,
    updated_at,
    event_type,
    payload,
    processed_at,
    attempts,
    max_attempts,
    next_attempt_at,
    last_error
FROM
    outbox_events
WHERE
    id = $1;

-- ==========================================
-- UPDATE
-- ==========================================
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
    next_attempt_at = CURRENT_TIMESTAMP +(INTERVAL '1 minute' * POWER(2, attempts + 1)::int)
WHERE
    id = $1;

-- name: OutboxEventResetAttempts :exec
UPDATE
    outbox_events
SET
    attempts = 0,
    next_attempt_at = CURRENT_TIMESTAMP,
    last_error = NULL
WHERE
    id = $1;

-- name: OutboxEventMarkDeadLetter :exec
-- Permanently ignores an event that cannot be processed.
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP,
    last_error = COALESCE($2, 'Manually marked dead letter by operator.'),
    attempts = max_attempts
WHERE
    id = $1;

-- ==========================================
-- DELETE
-- ==========================================
-- name: OutboxEventDeleteByID :exec
DELETE FROM outbox_events
WHERE id = $1;

-- name: OutboxEventPurgeProcessed :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - INTERVAL '7 days');

