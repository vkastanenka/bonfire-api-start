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