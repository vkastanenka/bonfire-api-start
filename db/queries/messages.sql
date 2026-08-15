-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, type, content, system_metadata)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('author_id')::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('forwarded_message_id')::uuid, sqlc.narg('forwarded_channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('content')::text, sqlc.narg('system_metadata')::jsonb)
RETURNING
    messages.*;

-- name: MessageGet :one
SELECT
    messages.*
FROM
    messages
WHERE
    id = @id::uuid;

-- name: MessageListAroundByChannelID :many
WITH before_read AS (
    SELECT
        id
    FROM
        messages
    WHERE
        channel_id = @channel_id::uuid
        AND id <= @last_read_message_id::uuid
    ORDER BY
        id DESC
    LIMIT @before_limit::int
),
after_read AS (
    SELECT
        id
    FROM
        messages
    WHERE
        channel_id = @channel_id::uuid
        AND id > @last_read_message_id::uuid
    ORDER BY
        id ASC
    LIMIT @after_limit::int
),
target_ids AS (
    SELECT
        id
    FROM
        before_read
    UNION ALL
    SELECT
        id
    FROM
        after_read
)
SELECT
    messages.*
FROM
    messages
    JOIN target_ids ON messages.id = target_ids.id
ORDER BY
    messages.id ASC;

-- name: MessageListBeforeByChannelID :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
    AND id < @before_id::uuid
ORDER BY
    id DESC
LIMIT @limit_val::int;

-- name: MessageListAfterByChannelID :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
    AND id > @after_id::uuid
ORDER BY
    id ASC
LIMIT @limit_val::int;

-- name: MessageListPinnedByChannelID :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
    AND pinned_at IS NOT NULL
ORDER BY
    pinned_at DESC,
    id DESC
LIMIT @limit_val::int;

-- name: MessageUpdateContent :one
UPDATE
    messages
SET
    content = @content::text,
    edited_at = @edited_at::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    messages.*;

-- name: MessageUpdatePinnedAt :one
UPDATE
    messages
SET
    pinned_at = sqlc.narg('pinned_at')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    messages.*;

-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = @id::uuid;

