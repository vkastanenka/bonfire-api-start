-- name: OutboxEventClaimPending :many
WITH target_events AS (
    SELECT
        id
    FROM
        outbox_events
    WHERE
        processed_at IS NULL
        AND attempts < max_attempts
        AND next_attempt_at <= @now::timestamptz
        AND (lease_expires_at IS NULL
            OR lease_expires_at < @now::timestamptz)
    ORDER BY
        next_attempt_at ASC,
        id ASC
    LIMIT @limit_val::int
    FOR UPDATE
        SKIP LOCKED)
UPDATE
    outbox_events o
SET
    locked_by = @worker_id::uuid,
    lease_expires_at = @lease_expires_at::timestamptz,
    updated_at = @now::timestamptz
FROM
    target_events t
WHERE
    o.id = t.id
RETURNING
    o.*;

-- name: OutboxEventCreate :exec
INSERT INTO outbox_events(id, aggregate_id, aggregate_type, event_type, payload, trace_id, created_at, updated_at, next_attempt_at, attempts, max_attempts)
    VALUES (@id::uuid, sqlc.narg('aggregate_id')::uuid, sqlc.narg('aggregate_type')::text, @event_type::text, @payload::jsonb, sqlc.narg('trace_id')::text, @created_at::timestamptz, @updated_at::timestamptz, @next_attempt_at::timestamptz, @attempts::int, @max_attempts::int);

-- name: OutboxEventCreateBatch :copyfrom
INSERT INTO outbox_events(id, aggregate_id, aggregate_type, event_type, payload, trace_id, created_at, updated_at, next_attempt_at, attempts, max_attempts)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: OutboxEventMarkProcessed :exec
UPDATE
    outbox_events
SET
    processed_at = @processed_at::timestamptz,
    locked_by = NULL,
    lease_expires_at = NULL,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL;

-- name: OutboxEventMarkDeadLetter :exec
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    next_attempt_at = @next_attempt_at::timestamptz,
    last_error = sqlc.narg('last_error')::text,
    locked_by = NULL,
    lease_expires_at = NULL,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL;

-- name: OutboxEventRenewLease :exec
UPDATE
    outbox_events
SET
    lease_expires_at = @lease_expires_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL;

-- name: OutboxEventReleaseLease :exec
UPDATE
    outbox_events
SET
    locked_by = NULL,
    lease_expires_at = NULL,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND locked_by = @worker_id::uuid
    AND processed_at IS NULL;

-- name: OutboxEventDeleteProcessedBatch :execrows
WITH targets AS (
    SELECT
        id
    FROM
        outbox_events
    WHERE
        processed_at IS NOT NULL
        AND processed_at < @before::timestamptz
    ORDER BY
        processed_at ASC,
        id ASC
    LIMIT @limit_val::int
    FOR UPDATE
        SKIP LOCKED)
DELETE FROM outbox_events o USING targets t
WHERE o.id = t.id;

