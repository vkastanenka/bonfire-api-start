-- name: OutboxEventCreate :one
INSERT INTO outbox_events(event_type, payload)
    VALUES ($1, $2)
RETURNING
    *;

-- name: OutboxEventGet :one
SELECT
    *
FROM
    outbox_events
WHERE
    id = $1;

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

-- name: OutboxEventDeleteOld :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - INTERVAL '7 days');

-- name: OutboxEventCountPending :one
SELECT
    COUNT(*)
FROM
    outbox_events
WHERE
    processed_at IS NULL;

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

