-- outbox_events
-- name: CreateOutboxEvent :one
INSERT INTO
    outbox_events (event_type, payload)
VALUES
    ($ 1, $ 2) RETURNING id,
    event_type,
    payload,
    created_at,
    processed_at,
    last_error,
    attempts,
    max_attempts,
    next_attempt_at;

-- name: GetUnprocessedOutboxEvents :many
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

-- name: MarkOutboxEventProcessed :exec
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP
WHERE
    id = $ 1;

-- name: RecordOutboxEventFailure :exec
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    last_error = $ 2,
    next_attempt_at = $ 3
WHERE
    id = $ 1;

-- name: DeleteOldProcessedEvents :exec
-- Maintenance: Prune the outbox to keep it performant
DELETE FROM
    outbox_events
WHERE
    processed_at < (CURRENT_TIMESTAMP - INTERVAL '7 days');

-- sessions
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

-- users
-- name: CreateUser :one
INSERT INTO
    users (email, username, password_hash, flags)
VALUES
    ($ 1, $ 2, $ 3, $ 4) RETURNING id,
    created_at,
    email,
    username,
    flags;

-- name: GetUserByID :one
SELECT
    id,
    created_at,
    updated_at,
    verified_at,
    email,
    username
FROM
    users
WHERE
    id = $ 1
LIMIT
    1;

-- name: GetUserByEmail :one
SELECT
    id,
    created_at,
    updated_at,
    verified_at,
    last_verification_sent_at,
    email,
    username
FROM
    users
WHERE
    email = $ 1
LIMIT
    1;

-- name: GetUserByUsername :one
SELECT
    id,
    created_at,
    updated_at,
    verified_at,
    email,
    username
FROM
    users
WHERE
    username = $ 1
LIMIT
    1;

-- name: GetUserAuthCredentials :one
SELECT
    id,
    password_hash,
    is_totp_enabled
FROM
    users
WHERE
    email = $ 1
LIMIT
    1;

-- name: DeleteUser :exec
DELETE FROM
    users
WHERE
    id = $ 1;

-- name: ValidateUserCredentialsAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            email = $ 1
    ) AS email_available,
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            username = $ 2
    ) AS username_available;

-- name: VerifyUserEmail :exec
UPDATE
    users
SET
    verified_at = CURRENT_TIMESTAMP,
    flags = flags | $ 2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $ 1
    AND verified_at IS NULL;

-- name: UpdateUserPassword :exec
UPDATE
    users
SET
    password_hash = $ 2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $ 1;

-- name: UpdateUserLastVerificationSent :exec
UPDATE
    users
SET
    last_verification_sent_at = CURRENT_TIMESTAMP
WHERE
    id = $ 1;

-- name: EnableUserTOTP :exec
UPDATE
    users
SET
    totp_secret = $ 1,
    is_totp_enabled = TRUE
WHERE
    id = $ 2;

-- name: DisableUserTOTP :exec
UPDATE
    users
SET
    totp_secret = NULL,
    is_totp_enabled = FALSE
WHERE
    id = $ 1;

-- name: GetUserTOTPSecret :one
SELECT
    totp_secret
FROM
    users
WHERE
    id = $ 1
LIMIT
    1;

-- user_profiles
-- name: CreateUserProfile :one
INSERT INTO
    user_profiles (user_id, display_name)
VALUES
    ($ 1, $ 2) RETURNING user_id,
    created_at,
    display_name;

-- name: GetUserProfile :one
SELECT
    user_id,
    created_at,
    updated_at,
    display_name
FROM
    user_profiles
WHERE
    user_id = $ 1;