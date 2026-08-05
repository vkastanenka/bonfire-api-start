-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, created_at, updated_at, last_seen_at, expires_at, revoked_at, client_ip, refresh_token_hash, os, client, user_agent)
    VALUES (@id::uuid, @user_id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @last_seen_at::timestamptz, @expires_at::timestamptz, sqlc.narg('revoked_at')::timestamptz, @client_ip::inet, @refresh_token_hash::bytea, @os::text, @client::text, @user_agent::text)
RETURNING
    id, user_id, created_at, updated_at, last_seen_at, expires_at, revoked_at, client_ip, refresh_token_hash, os, client, user_agent;

-- name: SessionDeleteExpiredBatch :exec
WITH targets AS (
    SELECT
        id
    FROM
        sessions
    WHERE
        expires_at <= @current_time::timestamptz
    ORDER BY
        expires_at ASC
    LIMIT @batch_limit::int
    FOR UPDATE
        SKIP LOCKED)
DELETE FROM sessions s USING targets t
WHERE s.id = t.id;

-- name: SessionGet :one
SELECT
    id,
    user_id,
    created_at,
    updated_at,
    last_seen_at,
    expires_at,
    revoked_at,
    client_ip,
    refresh_token_hash,
    os,
    client,
    user_agent
FROM
    sessions
WHERE
    id = @id::uuid
LIMIT 1;

-- name: SessionGetByRefreshTokenHash :one
SELECT
    id,
    user_id,
    created_at,
    updated_at,
    last_seen_at,
    expires_at,
    revoked_at,
    client_ip,
    refresh_token_hash,
    os,
    client,
    user_agent
FROM
    sessions
WHERE
    refresh_token_hash = @refresh_token_hash::bytea
LIMIT 1;

-- name: SessionListActiveByUser :many
SELECT
    id,
    user_id,
    created_at,
    updated_at,
    last_seen_at,
    expires_at,
    revoked_at,
    client_ip,
    os,
    client,
    user_agent
FROM
    sessions
WHERE
    user_id = @user_id::uuid
    AND revoked_at IS NULL
    AND expires_at > CURRENT_TIMESTAMP
ORDER BY
    last_seen_at DESC
LIMIT @limit_val::int;

-- name: SessionRevoke :exec
UPDATE
    sessions
SET
    revoked_at = @revoked_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND revoked_at IS NULL;

-- name: SessionRevokeAllForUser :exec
UPDATE
    sessions
SET
    revoked_at = @revoked_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    user_id = @user_id::uuid
    AND revoked_at IS NULL;

-- name: SessionRotateRefreshToken :one
UPDATE
    sessions
SET
    refresh_token_hash = @new_refresh_token_hash::bytea,
    expires_at = @expires_at::timestamptz,
    last_seen_at = @last_seen_at::timestamptz,
    updated_at = @updated_at::timestamptz,
    client_ip = @client_ip::inet,
    user_agent = @user_agent::text
WHERE
    id = @id::uuid
    AND refresh_token_hash = @old_refresh_token_hash::bytea
    AND revoked_at IS NULL
    AND expires_at > CURRENT_TIMESTAMP
RETURNING
    id,
    user_id,
    created_at,
    updated_at,
    last_seen_at,
    expires_at,
    revoked_at,
    client_ip,
    refresh_token_hash,
    os,
    client,
    user_agent;

-- name: SessionTouch :one
UPDATE
    sessions
SET
    last_seen_at = @last_seen_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND revoked_at IS NULL
    AND expires_at > CURRENT_TIMESTAMP
RETURNING
    id,
    last_seen_at,
    updated_at;

