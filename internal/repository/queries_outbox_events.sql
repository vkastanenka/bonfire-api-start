-- name: OutboxEventCreate :one
INSERT INTO
    outbox_events (event_type, payload)
VALUES
    ($ 1, $ 2) RETURNING *;

-- name: OutboxEventGet :one
SELECT
    *
FROM
    outbox_events
WHERE
    id = $ 1;

-- name: OutboxEventListUnprocessed :many
SELECT
    id,
    event_type,
    payload,
    attempts
FROM
    outbox_events
WHERE
    processed_at IS NULL
    AND attempts < max_attempts
    AND next_attempt_at <= CURRENT_TIMESTAMP
ORDER BY
    created_at ASC
LIMIT
    $ 1 FOR
UPDATE
    SKIP LOCKED;

-- name: OutboxEventMarkProcessed :exec
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP
WHERE
    id = $ 1;

-- name: OutboxEventRecordFailure :exec
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    last_error = $ 2,
    -- Exponential backoff: 2^attempts minutes
    next_attempt_at = CURRENT_TIMESTAMP + (INTERVAL '1 minute' * POWER(2, attempts + 1))
WHERE
    id = $ 1;

-- name: OutboxEventResetAttempts :exec
UPDATE
    outbox_events
SET
    attempts = 0,
    next_attempt_at = CURRENT_TIMESTAMP
WHERE
    id = $ 1;

-- name: OutboxEventDeleteOld :exec
DELETE FROM
    outbox_events
WHERE
    processed_at < (CURRENT_TIMESTAMP - INTERVAL '7 days');

-- name: OutboxEventCountPending :one
SELECT
    COUNT(*)
FROM
    outbox_events
WHERE
    processed_at IS NULL;