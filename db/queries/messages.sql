-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, type, content, system_metadata)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('author_id')::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('forwarded_message_id')::uuid, sqlc.narg('forwarded_channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('content')::text, sqlc.narg('system_metadata')::jsonb)
RETURNING
    messages.*;

-- name: MessageCreateBatch :many
WITH unpacked AS (
    SELECT
        x.*
    FROM
        jsonb_to_recordset(@payload::jsonb)
        WITH ORDINALITY AS x(id uuid, channel_id uuid, author_id uuid, reply_to_message_id uuid, forwarded_message_id uuid, forwarded_channel_id uuid, created_at timestamptz, updated_at timestamptz, type smallint, content text, system_metadata jsonb, ord bigint))
    INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, type, content, system_metadata)
    SELECT
        id,
        channel_id,
        author_id,
        reply_to_message_id,
        forwarded_message_id,
        forwarded_channel_id,
        created_at,
        updated_at,
        type,
        content,
        system_metadata
    FROM
        unpacked
    ORDER BY
        ord ASC
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
    AND (@cursor_id::uuid IS NULL
        OR (pinned_at,
            id) <(@cursor_pinned_at::timestamptz,
            @cursor_id::uuid))
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

