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

-- name: UserGetByUsername :one
SELECT
    users.*
FROM
    users
WHERE
    username = $1;

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
UPDATE
    users AS u
SET
    email = data.email,
    username = data.username,
    display_name = data.display_name,
    password_hash = data.password_hash,
    phone = data.phone,
    bio = data.bio,
    avatar_url = data.avatar_url,
    banner_color = data.banner_color,
    preferred_presence = data.preferred_presence,
    preferred_presence_until = data.preferred_presence_until,
    verified_at = data.verified_at,
    disabled_at = data.disabled_at,
    delete_scheduled_at = data.delete_scheduled_at,
    updated_at = data.updated_at
FROM (
    SELECT
        UNNEST(@ids::uuid[]) AS id,
        UNNEST(@emails::citext[]) AS email,
        UNNEST(@usernames::citext[]) AS username,
        UNNEST(@display_names::citext[]) AS display_name,
        UNNEST(@password_hashes::text[]) AS password_hash,
        UNNEST(@phones::text[]) AS phone,
        UNNEST(@bios::text[]) AS bio,
        UNNEST(@avatar_urls::text[]) AS avatar_url,
        UNNEST(@banner_colors::text[]) AS banner_color,
        UNNEST(@preferred_presences::smallint[]) AS preferred_presence,
        UNNEST(@preferred_presence_untils::timestamptz[]) AS preferred_presence_until,
        UNNEST(@verified_ats::timestamptz[]) AS verified_at,
        UNNEST(@disabled_ats::timestamptz[]) AS disabled_at,
        UNNEST(@delete_scheduled_ats::timestamptz[]) AS delete_scheduled_at,
        UNNEST(@updated_ats::timestamptz[]) AS updated_at) AS data
WHERE
    u.id = data.id
RETURNING
    u.*;

