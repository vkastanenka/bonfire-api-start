-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forward_message_id, forward_channel_id, created_at, updated_at, edited_at, pinned_at, type, content, metadata)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('author_id')::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('forward_message_id')::uuid, sqlc.narg('forward_channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, sqlc.narg('edited_at')::timestamptz, sqlc.narg('pinned_at')::timestamptz, @type::smallint, sqlc.narg('content')::text, sqlc.narg('metadata')::jsonb)
RETURNING
    messages.*;

-- name: MessageCreateBatch :many
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forward_message_id, forward_channel_id, created_at, updated_at, edited_at, pinned_at, type, content, metadata)
SELECT
    id,
    channel_id,
    author_id,
    reply_to_message_id,
    forward_message_id,
    forward_channel_id,
    created_at,
    updated_at,
    edited_at,
    pinned_at,
    type,
    content,
    metadata
FROM
    jsonb_to_recordset(@payload::jsonb) AS x(id uuid,
        channel_id uuid,
        author_id uuid,
        reply_to_message_id uuid,
        forward_message_id uuid,
        forward_channel_id uuid,
        created_at timestamptz,
        updated_at timestamptz,
        edited_at timestamptz,
        pinned_at timestamptz,
        type smallint,
        content text,
        metadata jsonb)
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
    AND id < @cursor_id::uuid
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
    AND id > @cursor_id::uuid
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
    AND (sqlc.narg('cursor_id')::uuid IS NULL
        OR (pinned_at,
            id) <(sqlc.narg('cursor_pinned_at')::timestamptz,
            sqlc.narg('cursor_id')::uuid))
ORDER BY
    pinned_at DESC,
    id DESC
LIMIT @limit_val::int;

-- name: MessageCountByChannelID :one
SELECT
    COUNT(*)::bigint
FROM
    messages
WHERE
    channel_id = @channel_id::uuid;

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

