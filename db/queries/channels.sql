-- name: ChannelCreate :one
INSERT INTO channels(id, created_at, updated_at, type, name, icon_url)
    VALUES (@id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('name')::text, sqlc.narg('icon_url')::text)
RETURNING
    channels.*;

-- name: ChannelGet :one
SELECT
    channels.*
FROM
    channels
WHERE
    id = @id::uuid;

-- name: ChannelGetForUpdate :one
SELECT
    channels.*
FROM
    channels
WHERE
    id = @id::uuid
FOR UPDATE;

-- name: ChannelGetBatch :many
SELECT
    channels.*
FROM
    channels
WHERE
    id = ANY (@ids::uuid[]);

-- name: ChannelUpdateGroup :one
UPDATE
    channels
SET
    name = sqlc.narg('name')::text,
    icon_url = sqlc.narg('icon_url')::text,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
    AND type = 2
RETURNING
    channels.*;

-- name: ChannelUpdateLastMessage :one
UPDATE
    channels
SET
    last_message_id = @last_message_id::uuid,
    last_message_at = @last_message_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    channels.*;

-- name: ChannelDelete :exec
DELETE FROM channels
WHERE id = @id::uuid;

