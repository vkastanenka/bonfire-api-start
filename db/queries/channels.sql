-- name: ChannelCreate :one
INSERT INTO channels(id, created_at, updated_at, type, name, icon_url)
    VALUES (@id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('name')::text, sqlc.narg('icon_url')::text)
RETURNING
    id, created_at, updated_at, type, name, icon_url;

-- name: ChannelDelete :exec
DELETE FROM channels
WHERE id = @id::uuid;

-- name: ChannelGet :one
SELECT
    id,
    created_at,
    updated_at,
    type,
    name,
    icon_url
FROM
    channels
WHERE
    id = @id::uuid;

-- name: ChannelUpdate :one
UPDATE
    channels
SET
    name = COALESCE(sqlc.narg('name')::text, name),
    icon_url = COALESCE(sqlc.narg('icon_url')::text, icon_url),
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    id,
    created_at,
    updated_at,
    type,
    name,
    icon_url;

