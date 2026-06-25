-- ==========================================
-- META
-- ==========================================
-- name: UserCount :one
SELECT
    COUNT(*)
FROM
    users;

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

-- ==========================================
-- CREATE
-- ==========================================
-- name: UserCreate :one
INSERT INTO users(email, username, password_hash)
    VALUES ($1, $2, $3)
RETURNING
    *;

-- ==========================================
-- LIST
-- ==========================================
-- name: UserList :many
SELECT
    *
FROM
    users
WHERE ($1::uuid IS NULL
    OR id < $1)
ORDER BY
    id DESC
LIMIT $2;

-- name: UserListUnverified :many
SELECT
    *
FROM
    users
WHERE
    verified_at IS NULL
ORDER BY
    created_at ASC
LIMIT $1;

-- ==========================================
-- GET
-- ==========================================
-- name: UserGetByID :one
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

-- ==========================================
-- UPDATE
-- ==========================================
-- name: UserMarkVerified :one
UPDATE
    users
SET
    verified_at = CURRENT_TIMESTAMP
WHERE
    id = $1
    AND verified_at IS NULL
RETURNING
    *;

-- name: UserUpdatePassword :one
UPDATE
    users
SET
    password_hash = $2,
    security_version = security_version + 1
WHERE
    id = $1
RETURNING
    *;

-- name: UserUpdateLastVerificationSent :one
UPDATE
    users
SET
    last_verification_sent_at = CURRENT_TIMESTAMP
WHERE
    id = $1
RETURNING
    *;

-- name: UserEnableTOTP :one
UPDATE
    users
SET
    totp_secret = $1,
    is_totp_enabled = TRUE
WHERE
    id = $2
RETURNING
    *;

-- name: UserDisableTOTP :one
UPDATE
    users
SET
    totp_secret = NULL,
    is_totp_enabled = FALSE
WHERE
    id = $1
RETURNING
    *;

-- ==========================================
-- DELETE
-- ==========================================
-- name: UserDeleteByID :exec
DELETE FROM users
WHERE id = $1;

-- name: UserDeleteByEmail :exec
DELETE FROM users
WHERE email = $1;

