-- name: UserCreate :one
INSERT INTO users(email, username, password_hash, role)
    VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- name: UserGet :one
SELECT
    *
FROM
    users
WHERE
    id = $1
LIMIT 1;

-- name: UserGetByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1
LIMIT 1;

-- name: UserGetByUsername :one
SELECT
    *
FROM
    users
WHERE
    username = $1
LIMIT 1;

-- name: UserGetAuthCredentials :one
SELECT
    id,
    password_hash,
    is_totp_enabled
FROM
    users
WHERE
    email = $1
LIMIT 1;

-- name: UserGetTOTPSecret :one
SELECT
    totp_secret
FROM
    users
WHERE
    id = $1
LIMIT 1;

-- name: UserListUnverified :many
SELECT
    id,
    email,
    username,
    created_at
FROM
    users
WHERE
    verified_at IS NULL
ORDER BY
    created_at ASC;

-- name: UserCheckAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            users u
        WHERE
            u.email = $1) AS email_available,
    NOT EXISTS (
        SELECT
            1
        FROM
            users u
        WHERE
            u.username = $2) AS username_available;

-- name: UserMarkVerified :exec
UPDATE
    users
SET
    verified_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $1
    AND verified_at IS NULL;

-- name: UserUpdatePassword :exec
UPDATE
    users
SET
    password_hash = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: UserUpdateLastVerificationSent :exec
UPDATE
    users
SET
    last_verification_sent_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: UserEnableTOTP :exec
UPDATE
    users
SET
    totp_secret = $1,
    is_totp_enabled = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $2;

-- name: UserDisableTOTP :exec
UPDATE
    users
SET
    totp_secret = NULL,
    is_totp_enabled = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = $1;

-- name: UserDelete :exec
DELETE FROM users
WHERE id = $1;

-- name: UserDeleteByEmail :one
DELETE FROM users
WHERE email = $1
RETURNING
    *;

