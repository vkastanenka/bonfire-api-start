-- name: UserAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            email = @email::citext)::boolean AS email_available,
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            username = @username::citext)::boolean AS username_available;

-- name: UserCreate :exec
INSERT INTO users(id, email, username, display_name, password_hash, phone, bio, avatar_url, banner_color, preferred_presence, preferred_presence_until, verified_at, disabled_at, delete_scheduled_at, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING
    users.*;

-- name: UserGet :one
SELECT
    users.*
FROM
    users
WHERE
    id = $1;

-- name: UserGetByEmail :one
SELECT
    users.*
FROM
    users
WHERE
    email = $1;

-- name: UserListDeleteScheduled :many
SELECT
    users.*
FROM
    users
WHERE
    delete_scheduled_at IS NOT NULL
    AND delete_scheduled_at <= @now::timestamptz
ORDER BY
    delete_scheduled_at ASC
LIMIT @batch_limit::int;

-- name: UserUpdate :one
UPDATE
    users
SET
    email = $2,
    username = $3,
    display_name = $4,
    password_hash = $5,
    phone = $6,
    bio = $7,
    avatar_url = $8,
    banner_color = $9,
    preferred_presence = $10,
    preferred_presence_until = $11,
    verified_at = $12,
    disabled_at = $13,
    delete_scheduled_at = $14,
    updated_at = $15
WHERE
    id = $1
RETURNING
    users.*;

-- name: UserUpdateBatch :many
WITH input_data AS (
    SELECT
        *
    FROM
        jsonb_populate_recordset(NULL::users, @users_json::jsonb))
INSERT INTO users(id, email, username, display_name, password_hash, phone, bio, avatar_url, banner_color, preferred_presence, preferred_presence_until, verified_at, disabled_at, delete_scheduled_at, created_at, updated_at)
SELECT
    id,
    email,
    username,
    display_name,
    password_hash,
    phone,
    bio,
    avatar_url,
    banner_color,
    preferred_presence,
    preferred_presence_until,
    verified_at,
    disabled_at,
    delete_scheduled_at,
    created_at,
    updated_at
FROM
    input_data
ORDER BY
    id ASC
ON CONFLICT (id)
    DO UPDATE SET
        email = EXCLUDED.email,
        username = EXCLUDED.username,
        display_name = EXCLUDED.display_name,
        password_hash = EXCLUDED.password_hash,
        phone = EXCLUDED.phone,
        bio = EXCLUDED.bio,
        avatar_url = EXCLUDED.avatar_url,
        banner_color = EXCLUDED.banner_color,
        preferred_presence = EXCLUDED.preferred_presence,
        preferred_presence_until = EXCLUDED.preferred_presence_until,
        verified_at = EXCLUDED.verified_at,
        disabled_at = EXCLUDED.disabled_at,
        delete_scheduled_at = EXCLUDED.delete_scheduled_at,
        updated_at = EXCLUDED.updated_at
    RETURNING
        users.*;

