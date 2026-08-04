-- name: UserCheckAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            users.email = $1)::boolean AS email_available,
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            users.username = $2)::boolean AS username_available;

-- name: UserCreateAggregate :exec
WITH new_user AS (
INSERT INTO users(id, created_at, updated_at, verified_at, disabled_at, delete_scheduled_at, preferred_presence_until, preferred_presence, email, username, password_hash, phone)
        VALUES (@id::uuid, @created_at::timestamptz, @updated_at::timestamptz, sqlc.narg('verified_at')::timestamptz, sqlc.narg('disabled_at')::timestamptz, sqlc.narg('delete_scheduled_at')::timestamptz, sqlc.narg('preferred_presence_until')::timestamptz, sqlc.narg('preferred_presence')::smallint, @email::citext, @username::citext, @password_hash::text, sqlc.narg('phone')::text)
    RETURNING
        id
), new_profile AS (
INSERT INTO user_profiles(user_id, created_at, updated_at, display_name, bio, avatar_url, banner_color)
        VALUES ((
                SELECT
                    id
                FROM
                    new_user),
                @profile_created_at::timestamptz,
                @profile_updated_at::timestamptz,
                @display_name::citext,
                sqlc.narg('bio')::text,
                sqlc.narg('avatar_url')::text,
                sqlc.narg('banner_color')::text)
        RETURNING
            user_id);

-- name: UserGet :one
SELECT
    id,
    email,
    username,
    password_hash,
    phone,
    preferred_presence,
    preferred_presence_until,
    verified_at,
    disabled_at,
    delete_scheduled_at,
    created_at,
    updated_at
FROM
    users
WHERE
    id = @id::uuid
LIMIT 1;

-- name: UserGetAggregate :one
SELECT
    id,
    email,
    username,
    phone,
    display_name,
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
    user_aggregates
WHERE
    id = @id::uuid
LIMIT 1;

-- name: UserGetByEmail :one
SELECT
    id,
    email,
    username,
    password_hash,
    phone,
    preferred_presence,
    preferred_presence_until,
    verified_at,
    disabled_at,
    delete_scheduled_at,
    created_at,
    updated_at
FROM
    users
WHERE
    email = @email::citext
LIMIT 1;

-- name: UserGetByPhone :one
SELECT
    id,
    email,
    username,
    password_hash,
    phone,
    preferred_presence,
    preferred_presence_until,
    verified_at,
    disabled_at,
    delete_scheduled_at,
    created_at,
    updated_at
FROM
    users
WHERE
    phone = @phone::text
LIMIT 1;

-- name: UserGetByUsername :one
SELECT
    id,
    email,
    username,
    password_hash,
    phone,
    preferred_presence,
    preferred_presence_until,
    verified_at,
    disabled_at,
    delete_scheduled_at,
    created_at,
    updated_at
FROM
    users
WHERE
    username = @username::citext
LIMIT 1;

-- name: UserProfileUpsert :one
INSERT INTO user_profiles(user_id, created_at, updated_at, display_name, bio, avatar_url, banner_color)
    VALUES (@user_id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @display_name::citext, sqlc.narg('bio')::text, sqlc.narg('avatar_url')::text, sqlc.narg('banner_color')::text)
ON CONFLICT (user_id)
    DO UPDATE SET
        display_name = EXCLUDED.display_name,
        bio = EXCLUDED.bio,
        avatar_url = EXCLUDED.avatar_url,
        banner_color = EXCLUDED.banner_color,
        updated_at = EXCLUDED.updated_at
    RETURNING
        user_id,
        created_at,
        updated_at,
        display_name,
        bio,
        avatar_url,
        banner_color;

-- name: UserUpdate :one
UPDATE
    users
SET
    email = @email::citext,
    username = @username::citext,
    password_hash = @password_hash::text,
    phone = sqlc.narg('phone')::text,
    preferred_presence = sqlc.narg('preferred_presence')::smallint,
    preferred_presence_until = sqlc.narg('preferred_presence_until')::timestamptz,
    verified_at = sqlc.narg('verified_at')::timestamptz,
    disabled_at = sqlc.narg('disabled_at')::timestamptz,
    delete_scheduled_at = sqlc.narg('delete_scheduled_at')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    id,
    email,
    username,
    password_hash,
    phone,
    preferred_presence,
    preferred_presence_until,
    verified_at,
    disabled_at,
    delete_scheduled_at,
    created_at,
    updated_at;

