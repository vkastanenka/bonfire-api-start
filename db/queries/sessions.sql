-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, created_at, updated_at, last_seen_at, expires_at, revoked_at, client_ip, refresh_token_hash, os, client, user_agent)
    VALUES (@id::uuid, @user_id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @last_seen_at::timestamptz, @expires_at::timestamptz, sqlc.narg('revoked_at')::timestamptz, @client_ip::inet, @refresh_token_hash::bytea, @os::text, @client::text, @user_agent::text)
RETURNING
    id, user_id, created_at, updated_at, last_seen_at, expires_at, revoked_at, client_ip, refresh_token_hash, os, client, user_agent;

-- name: SessionGet :one
SELECT
    sessions.*
FROM
    sessions
WHERE
    id = @id::uuid;

-- name: SessionListValidByUserID :many
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

-- name: SessionRotateRefreshTokenHash :one
UPDATE
    sessions
SET
    refresh_token_hash = @new_refresh_token_hash::bytea,
    expires_at = @expires_at::timestamptz,
    last_seen_at = @now::timestamptz,
    updated_at = @now::timestamptz,
    client_ip = @client_ip::inet,
    user_agent = @user_agent::text
WHERE
    id = @id::uuid
    AND refresh_token_hash = @old_refresh_token_hash::bytea
    AND revoked_at IS NULL
    AND expires_at > @now::timestamptz
RETURNING
    sessions.*;

-- name: SessionRevoke :exec
UPDATE
    sessions
SET
    revoked_at = @now::timestamptz,
    updated_at = @now::timestamptz
WHERE
    id = @id::uuid
    AND user_id = @user_id::uuid
    AND revoked_at IS NULL;

-- name: SessionRevokeAll :exec
UPDATE
    sessions
SET
    revoked_at = @now::timestamptz,
    updated_at = @now::timestamptz
WHERE
    user_id = @user_id::uuid
    AND revoked_at IS NULL;

-- name: SessionDeleteBatchExpired :exec
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
    LIMIT @limit_val::int
    FOR UPDATE
        SKIP LOCKED)
DELETE FROM sessions
WHERE id IN (
        SELECT
            id
        FROM
            targets);

