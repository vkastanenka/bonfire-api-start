-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, refresh_token_hash, client_ip, user_agent, os, client, expires_at, last_seen_at, revoked_at, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING
    sessions.*;

-- name: SessionDeleteExpiredBatch :exec
WITH targets AS (
    SELECT
        id
    FROM
        sessions
    WHERE
        expires_at <= @now::timestamptz
    ORDER BY
        expires_at ASC,
        id ASC
    LIMIT @batch_limit::int
    FOR UPDATE
        SKIP LOCKED)
DELETE FROM sessions s USING targets t
WHERE s.id = t.id;

-- name: SessionGet :one
SELECT
    sessions.*
FROM
    sessions
WHERE
    id = $1;

-- name: SessionGetByRefreshTokenHash :one
SELECT
    sessions.*
FROM
    sessions
WHERE
    refresh_token_hash = $1;

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
    AND expires_at > @now::timestamptz
RETURNING
    sessions.*;

-- name: SessionUpdate :one
UPDATE
    sessions
SET
    user_id = $2,
    refresh_token_hash = $3,
    client_ip = $4,
    user_agent = $5,
    os = $6,
    client = $7,
    expires_at = $8,
    last_seen_at = $9,
    revoked_at = $10,
    updated_at = $11
WHERE
    id = $1
RETURNING
    sessions.*;

-- name: SessionUpdateBatch :many
WITH input_data AS (
    SELECT
        *
    FROM
        jsonb_populate_recordset(NULL::sessions, @sessions_json::jsonb))
INSERT INTO sessions(id, user_id, refresh_token_hash, client_ip, user_agent, os, client, expires_at, last_seen_at, revoked_at, created_at, updated_at)
SELECT
    id,
    user_id,
    refresh_token_hash,
    client_ip,
    user_agent,
    os,
    client,
    expires_at,
    last_seen_at,
    revoked_at,
    created_at,
    updated_at
FROM
    input_data
ORDER BY
    id ASC
ON CONFLICT (id)
    DO UPDATE SET
        user_id = EXCLUDED.user_id,
        refresh_token_hash = EXCLUDED.refresh_token_hash,
        client_ip = EXCLUDED.client_ip,
        user_agent = EXCLUDED.user_agent,
        os = EXCLUDED.os,
        client = EXCLUDED.client,
        expires_at = EXCLUDED.expires_at,
        last_seen_at = EXCLUDED.last_seen_at,
        revoked_at = EXCLUDED.revoked_at,
        updated_at = EXCLUDED.updated_at
    RETURNING
        sessions.*;

-- name: SessionUserGetBatch :many
SELECT
    sessions.*
FROM
    sessions
WHERE
    user_id = @user_id::uuid
    AND revoked_at IS NULL
    AND expires_at > @now::timestamptz
ORDER BY
    last_seen_at DESC
LIMIT @limit_val::int;

-- name: SessionUserRevoke :exec
UPDATE
    sessions
SET
    revoked_at = @revoked_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND user_id = @user_id::uuid
    AND revoked_at IS NULL;

-- name: SessionUserRevokeAll :exec
UPDATE
    sessions
SET
    revoked_at = @revoked_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    user_id = @user_id::uuid
    AND revoked_at IS NULL;

