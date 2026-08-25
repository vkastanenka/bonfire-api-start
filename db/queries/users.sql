-- name: UserCreate :one
INSERT INTO users(id, email, username, display_name, password_hash, phone, bio, avatar_url, banner_color, preferred_presence, preferred_presence_until, verified_at, disabled_at, delete_scheduled_at, created_at, updated_at)
    VALUES (@id::uuid, @email::citext, @username::citext, @display_name::citext, @password_hash::text, sqlc.narg('phone')::text, sqlc.narg('bio')::text, sqlc.narg('avatar_url')::text, sqlc.narg('banner_color')::text, sqlc.narg('preferred_presence')::smallint, sqlc.narg('preferred_presence_until')::timestamptz, sqlc.narg('verified_at')::timestamptz, sqlc.narg('disabled_at')::timestamptz, sqlc.narg('delete_scheduled_at')::timestamptz, @created_at::timestamptz, @updated_at::timestamptz)
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

-- name: UserGetBatch :many
SELECT
    users.*
FROM
    users
WHERE
    id = ANY (@ids::uuid[]);

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
LIMIT @limit_val::int
FOR UPDATE
    SKIP LOCKED;

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

-- name: UserUpdate :one
UPDATE
    users
SET
    email = @email::citext,
    username = @username::citext,
    display_name = @display_name::citext,
    password_hash = @password_hash::text,
    phone = sqlc.narg('phone')::text,
    bio = sqlc.narg('bio')::text,
    avatar_url = sqlc.narg('avatar_url')::text,
    banner_color = sqlc.narg('banner_color')::text,
    preferred_presence = sqlc.narg('preferred_presence')::smallint,
    preferred_presence_until = sqlc.narg('preferred_presence_until')::timestamptz,
    verified_at = sqlc.narg('verified_at')::timestamptz,
    disabled_at = sqlc.narg('disabled_at')::timestamptz,
    delete_scheduled_at = sqlc.narg('delete_scheduled_at')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserUpdateEmail :one
UPDATE
    users
SET
    email = @email::citext,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserUpdateUsername :one
UPDATE
    users
SET
    username = @username::citext,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserUpdatePhone :one
UPDATE
    users
SET
    phone = sqlc.narg('phone')::text,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserUpdatePasswordHash :one
UPDATE
    users
SET
    password_hash = @password_hash::text,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserUpdateProfile :one
UPDATE
    users
SET
    display_name = @display_name::citext,
    bio = sqlc.narg('bio')::text,
    avatar_url = sqlc.narg('avatar_url')::text,
    banner_color = sqlc.narg('banner_color')::text,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserUpdatePresence :one
UPDATE
    users
SET
    preferred_presence = sqlc.narg('preferred_presence')::smallint,
    preferred_presence_until = sqlc.narg('preferred_presence_until')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserVerify :one
UPDATE
    users
SET
    verified_at = @verified_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserSetDisabled :one
UPDATE
    users
SET
    disabled_at = sqlc.narg('disabled_at')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    users.*;

-- name: UserSetDeleteSchedule :one
UPDATE
    users
SET
    delete_scheduled_at = sqlc.narg('delete_scheduled_at')::timestamptz,
    disabled_at = sqlc.narg('disabled_at')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
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

