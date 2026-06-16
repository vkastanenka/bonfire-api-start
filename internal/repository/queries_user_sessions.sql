-- name: CreateSession :one
INSERT INTO
    sessions (
        user_id,
        refresh_token,
        user_agent,
        client_ip,
        is_blocked,
        expires_at
    )
VALUES
    ($ 1, $ 2, $ 3, $ 4, $ 5, $ 6) RETURNING id,
    user_id,
    refresh_token,
    user_agent,
    client_ip,
    is_blocked,
    expires_at,
    created_at;

-- name: GetSession :one
SELECT
    id,
    user_id,
    refresh_token,
    user_agent,
    client_ip,
    is_blocked,
    expires_at,
    created_at
FROM
    sessions
WHERE
    refresh_token = $ 1
LIMIT
    1;

-- name: UpdateSessionRefreshToken :exec
UPDATE
    sessions
SET
    refresh_token = $ 2,
    expires_at = $ 3
WHERE
    id = $ 1;

-- name: GetUserSessions :many
SELECT
    id,
    user_agent,
    client_ip,
    created_at,
    last_seen_at,
    refresh_token
FROM
    sessions
WHERE
    user_id = $ 1
    AND is_blocked = false;

-- name: DeleteSession :exec
DELETE FROM
    sessions
WHERE
    id = $ 1
    AND user_id = $ 2;

-- name: DeleteAllSessionsExcept :exec
DELETE FROM
    sessions
WHERE
    user_id = @user_id
    AND id != @id;