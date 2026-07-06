-- name: UserCreate :one
INSERT INTO users(id, email, username, password_hash)
    VALUES ($1, $2, $3, $4)
RETURNING
    *;

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

-- name: UserProfileCreate :one
INSERT INTO user_profiles(user_id, display_name)
    VALUES ($1, $2)
RETURNING
    *;

-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, refresh_token, expires_at)
    VALUES ($1, $2, $3, $4)
RETURNING
    *;

