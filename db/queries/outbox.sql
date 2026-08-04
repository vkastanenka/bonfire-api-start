-- name: OutboxEventCreate :one
INSERT INTO outbox_events(id, locked_by, created_at, updated_at, next_attempt_at, lease_expires_at, processed_at, attempts, max_attempts, event_type, last_error, payload)
    VALUES (@id, @locked_by, @created_at, @updated_at, @next_attempt_at, @lease_expires_at, @processed_at, @attempts, @max_attempts, @event_type, @last_error, @payload)
RETURNING
    *;

-- name: OutboxEventGet :one
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
WHERE (sqlc.narg('cursor_id')::uuid IS NULL
    OR id < sqlc.narg('cursor_id'))
ORDER BY
    id DESC
LIMIT @result_limit;

-- name: OutboxEventAcquireBatch :many
UPDATE
    outbox_events
SET
    locked_by = sqlc.arg(worker_id)::uuid,
    lease_expires_at = CURRENT_TIMESTAMP + make_interval(secs => sqlc.arg(lease_duration_seconds)::int),
    updated_at = CURRENT_TIMESTAMP
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
        LIMIT sqlc.arg(batch_size)::int
        FOR UPDATE
            SKIP LOCKED)
RETURNING
    *;

-- name: OutboxEventUpdate :one
UPDATE
    outbox_events
SET
    locked_by = sqlc.arg(locked_by),
    lease_expires_at = sqlc.arg(lease_expires_at),
    processed_at = sqlc.arg(processed_at),
    attempts = sqlc.arg(attempts),
    max_attempts = sqlc.arg(max_attempts),
    next_attempt_at = sqlc.arg(next_attempt_at),
    last_error = sqlc.arg(last_error),
    updated_at = sqlc.arg(updated_at)
WHERE
    id = sqlc.arg(id)
RETURNING
    *;

-- name: OutboxEventRenewLease :exec
UPDATE
    outbox_events
SET
    lease_expires_at = CURRENT_TIMESTAMP + make_interval(secs => sqlc.arg(lease_duration_seconds)::int),
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = sqlc.arg(id)::uuid
    AND locked_by = sqlc.arg(worker_id)::uuid
    AND processed_at IS NULL;

-- name: OutboxEventDelete :exec
DELETE FROM outbox_events
WHERE id = $1;

-- name: OutboxEventPurgeProcessed :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - make_interval(days => sqlc.arg(retention_days)::int));

