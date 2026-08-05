-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, edited_at, pinned_at, type, content, system_metadata)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('author_id')::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('forwarded_message_id')::uuid, sqlc.narg('forwarded_channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, sqlc.narg('edited_at')::timestamptz, sqlc.narg('pinned_at')::timestamptz, @type::smallint, sqlc.narg('content')::text, sqlc.narg('system_metadata')::jsonb)
RETURNING
    id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, edited_at, pinned_at, type, content, system_metadata;

-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = @id::uuid;

-- name: MessageGet :one
SELECT
    id,
    channel_id,
    author_id,
    reply_to_message_id,
    forwarded_message_id,
    forwarded_channel_id,
    created_at,
    updated_at,
    edited_at,
    pinned_at,
    type,
    content,
    system_metadata
FROM
    messages
WHERE
    id = @id::uuid;

-- name: MessageUpdateContent :one
UPDATE
    messages
SET
    content = COALESCE(sqlc.narg('content')::text, content),
    updated_at = @updated_at::timestamptz,
    edited_at = @edited_at::timestamptz
WHERE
    id = @id::uuid
    AND author_id = @author_id::uuid
RETURNING
    id,
    channel_id,
    author_id,
    reply_to_message_id,
    forwarded_message_id,
    forwarded_channel_id,
    created_at,
    updated_at,
    edited_at,
    pinned_at,
    type,
    content,
    system_metadata;

