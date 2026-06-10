-- users

-- name: CreateUser :one
INSERT INTO users (email, username, password_hash)
VALUES ($1, $2, $3)
RETURNING id, created_at, email, username;

-- name: GetUserByID :one
SELECT id, created_at, updated_at, deleted_at, verified_at, email, username
FROM users 
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, deleted_at, verified_at, email, username
FROM users 
WHERE email = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT id, created_at, updated_at, deleted_at, verified_at, email, username
FROM users
WHERE username = $1 LIMIT 1;

-- name: GetUserAuthCredentials :one
SELECT id, password_hash
FROM users 
WHERE email = $1 LIMIT 1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ValidateUserCredentialsAvailability :one
SELECT
    NOT EXISTS (
        SELECT 1
        FROM users u
        WHERE u.email = @email
          AND u.deleted_at IS NULL
    ) AS email_available,
    NOT EXISTS (
        SELECT 1
        FROM users u
        WHERE u.username = @username
          AND u.deleted_at IS NULL
    ) AS username_available;

-- user_profiles

-- name: CreateUserProfile :one
INSERT INTO user_profiles (user_id, display_name)
VALUES ($1, $2)
RETURNING user_id, created_at, display_name;

-- name: GetUserProfile :one
SELECT user_id, created_at, updated_at, display_name
FROM user_profiles
WHERE user_id = $1;