-- name: UserAggregateCreate :exec
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

-- name: UserAggregateGet :one
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

-- name: UserAggregateUpdateBatch :exec
WITH unnested_data AS (
    SELECT
        unnest(@ids::uuid[]) AS id,
        unnest(@emails::citext[]) AS email,
        unnest(@usernames::citext[]) AS username,
        unnest(@password_hashes::text[]) AS password_hash,
        unnest(@phones::text[]) AS phone,
        unnest(@preferred_presences::smallint[]) AS preferred_presence,
        unnest(@preferred_presence_untils::timestamptz[]) AS preferred_presence_until,
        unnest(@verified_ats::timestamptz[]) AS verified_at,
        unnest(@disabled_ats::timestamptz[]) AS disabled_at,
        unnest(@delete_scheduled_ats::timestamptz[]) AS delete_scheduled_at,
        unnest(@updated_ats::timestamptz[]) AS updated_at,
        unnest(@display_names::citext[]) AS display_name,
        unnest(@bios::text[]) AS bio,
        unnest(@avatar_urls::text[]) AS avatar_url,
        unnest(@banner_colors::text[]) AS banner_color,
        unnest(@profile_updated_ats::timestamptz[]) AS profile_updated_at
),
updated_users AS (
    UPDATE
        users u
    SET
        email = d.email,
        username = d.username,
        password_hash = d.password_hash,
        phone = d.phone,
        preferred_presence = d.preferred_presence,
        preferred_presence_until = d.preferred_presence_until,
        verified_at = d.verified_at,
        disabled_at = d.disabled_at,
        delete_scheduled_at = d.delete_scheduled_at,
        updated_at = d.updated_at
    FROM
        unnested_data d
    WHERE
        u.id = d.id
    RETURNING
        u.id
),
updated_profiles AS (
    UPDATE
        user_profiles p
    SET
        display_name = d.display_name,
        bio = d.bio,
        avatar_url = d.avatar_url,
        banner_color = d.banner_color,
        updated_at = d.profile_updated_at
    FROM
        unnested_data d
    WHERE
        p.user_id = d.id
        AND p.user_id IN (
            SELECT
                id
            FROM
                updated_users)
        RETURNING
            p.user_id);

-- name: UserAvailability :one
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

-- name: UserListDeleteScheduled :many
SELECT
    id,
    delete_scheduled_at
FROM
    users
WHERE
    delete_scheduled_at IS NOT NULL
    AND delete_scheduled_at <= @current_time::timestamptz
ORDER BY
    delete_scheduled_at ASC,
    id ASC
LIMIT @batch_limit::int;

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

