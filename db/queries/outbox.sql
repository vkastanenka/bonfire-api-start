-- name: OutboxEventAcquireBatch :many
UPDATE
    outbox_events
SET
    locked_by = @worker_id::uuid,
    lease_expires_at = CURRENT_TIMESTAMP + make_interval(secs => @lease_duration_seconds::int),
    updated_at = CURRENT_TIMESTAMP
WHERE (id, created_at) IN (
    SELECT
        id,
        created_at
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
    LIMIT @batch_size::int
    FOR UPDATE
        SKIP LOCKED)
RETURNING
    id,
    aggregate_id,
    aggregate_type,
    event_type,
    payload,
    trace_id,
    created_at,
    updated_at,
    next_attempt_at,
    lease_expires_at,
    processed_at,
    locked_by,
    attempts,
    max_attempts,
    last_error;

-- name: OutboxEventCreate :one
INSERT INTO outbox_events(id, aggregate_id, aggregate_type, event_type, payload, trace_id, created_at, updated_at, next_attempt_at, attempts, max_attempts)
    VALUES (@id::uuid, sqlc.narg('aggregate_id')::uuid, sqlc.narg('aggregate_type')::text, @event_type::text, @payload::jsonb, sqlc.narg('trace_id')::text, @created_at::timestamptz, @updated_at::timestamptz, @next_attempt_at::timestamptz, @attempts::int, @max_attempts::int)
RETURNING
    id, aggregate_id, aggregate_type, event_type, payload, trace_id, created_at, updated_at, next_attempt_at, lease_expires_at, processed_at, locked_by, attempts, max_attempts, last_error;

-- name: OutboxEventCreateBatch :copyfrom
INSERT INTO outbox_events(id, aggregate_id, aggregate_type, event_type, payload, trace_id, created_at, updated_at, next_attempt_at, attempts, max_attempts)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: OutboxEventGet :one
SELECT
    id,
    aggregate_id,
    aggregate_type,
    event_type,
    payload,
    trace_id,
    created_at,
    updated_at,
    next_attempt_at,
    lease_expires_at,
    processed_at,
    locked_by,
    attempts,
    max_attempts,
    last_error
FROM
    outbox_events
WHERE
    id = @id::uuid
    AND created_at = @created_at::timestamptz
LIMIT 1;

-- name: OutboxEventListDeadLetter :many
SELECT
    id,
    aggregate_id,
    aggregate_type,
    event_type,
    payload,
    trace_id,
    created_at,
    updated_at,
    next_attempt_at,
    attempts,
    max_attempts,
    last_error
FROM
    outbox_events
WHERE
    processed_at IS NULL
    AND attempts >= max_attempts
    AND (@cursor_id::uuid IS NULL
        OR id < @cursor_id::uuid)
ORDER BY
    id DESC
LIMIT @limit_val::int;

-- name: OutboxEventMarkFailed :one
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    next_attempt_at = @next_attempt_at::timestamptz,
    last_error = @last_error::text,
    locked_by = NULL,
    lease_expires_at = NULL,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND created_at = @created_at::timestamptz
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL
RETURNING
    id,
    created_at,
    attempts,
    next_attempt_at,
    last_error,
    updated_at;

-- name: OutboxEventMarkProcessed :one
UPDATE
    outbox_events
SET
    processed_at = @processed_at::timestamptz,
    locked_by = NULL,
    lease_expires_at = NULL,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND created_at = @created_at::timestamptz
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL
RETURNING
    id,
    created_at,
    processed_at,
    updated_at;

-- name: OutboxEventReleaseLease :exec
UPDATE
    outbox_events
SET
    locked_by = NULL,
    lease_expires_at = NULL,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND created_at = @created_at::timestamptz
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL;

-- name: OutboxEventRenewLease :exec
UPDATE
    outbox_events
SET
    lease_expires_at = CURRENT_TIMESTAMP + make_interval(secs => @lease_duration_seconds::int),
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = @id::uuid
    AND created_at = @created_at::timestamptz
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL;

