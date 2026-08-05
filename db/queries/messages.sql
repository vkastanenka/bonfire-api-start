-- name: MessageAuthorDelete :execrows
DELETE FROM messages
WHERE id = @message_id::uuid
    AND author_id = @user_id::uuid;

-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, edited_at, pinned_at, type, content, system_metadata)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('author_id')::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('forwarded_message_id')::uuid, sqlc.narg('forwarded_channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, sqlc.narg('edited_at')::timestamptz, sqlc.narg('pinned_at')::timestamptz, @type::smallint, sqlc.narg('content')::text, sqlc.narg('system_metadata')::jsonb)
RETURNING
    id, channel_id, author_id, reply_to_message_id, forwarded_message_id, forwarded_channel_id, created_at, updated_at, edited_at, pinned_at, type, content, system_metadata;

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

-- name: MessageListAggregateAfter :many
WITH user_access AS (
    SELECT
        1
    FROM
        channel_members
    WHERE
        channel_id = @channel_id::uuid
        AND user_id = @user_id::uuid
),
target_ids AS (
    SELECT
        m.id,
        m.created_at
    FROM
        messages m
        CROSS JOIN user_access
    WHERE
        m.channel_id = @channel_id::uuid
        AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
            OR (m.created_at,
                m.id) >(sqlc.narg('cursor_created_at')::timestamptz,
                sqlc.narg('cursor_id')::uuid))
    ORDER BY
        m.created_at ASC,
        m.id ASC
    LIMIT @limit_val::int
)
SELECT
    m.id,
    m.channel_id,
    m.type,
    m.content,
    m.system_metadata,
    m.author_id,
    m.reply_to_message_id,
    m.forwarded_message_id,
    m.forwarded_channel_id,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    target_ids ti
    JOIN messages m ON m.id = ti.id
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
            ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
        FROM
            message_attachments a
        WHERE
            a.message_id = ti.id) att ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
            ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
        FROM
            message_reactions r
        WHERE
            r.message_id = ti.id) rec ON TRUE
ORDER BY
    ti.created_at ASC,
    ti.id ASC;

-- name: MessageListAggregateAround :many
WITH user_access AS (
    SELECT
        1
    FROM
        channel_members
    WHERE
        channel_id = @channel_id::uuid
        AND user_id = @user_id::uuid
),
target AS (
    SELECT
        m.created_at
    FROM
        messages m
        CROSS JOIN user_access
    WHERE
        m.id = @target_id::uuid
        AND m.channel_id = @channel_id::uuid
),
older_window AS (
    SELECT
        m.id,
        m.created_at
    FROM
        messages m,
        target t
    WHERE
        m.channel_id = @channel_id::uuid
        AND (m.created_at,
            m.id) <=(t.created_at,
            @target_id::uuid)
    ORDER BY
        m.created_at DESC,
        m.id DESC
    LIMIT @older_limit::int
),
newer_window AS (
    SELECT
        m.id,
        m.created_at
    FROM
        messages m,
        target t
    WHERE
        m.channel_id = @channel_id::uuid
        AND (m.created_at,
            m.id) >(t.created_at,
            @target_id::uuid)
    ORDER BY
        m.created_at ASC,
        m.id ASC
    LIMIT @newer_limit::int
),
around_ids AS (
    SELECT
        id,
        created_at
    FROM
        older_window
    UNION ALL
    SELECT
        id,
        created_at
    FROM
        newer_window
)
SELECT
    m.id,
    m.channel_id,
    m.type,
    m.content,
    m.system_metadata,
    m.author_id,
    m.reply_to_message_id,
    m.forwarded_message_id,
    m.forwarded_channel_id,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    around_ids ai
    JOIN messages m ON m.id = ai.id
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
            ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
        FROM
            message_attachments a
        WHERE
            a.message_id = ai.id) att ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
            ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
        FROM
            message_reactions r
        WHERE
            r.message_id = ai.id) rec ON TRUE
ORDER BY
    ai.created_at ASC,
    ai.id ASC;

-- name: MessageListAggregateBefore :many
WITH user_access AS (
    SELECT
        1
    FROM
        channel_members
    WHERE
        channel_id = @channel_id::uuid
        AND user_id = @user_id::uuid
),
target_ids AS (
    SELECT
        m.id,
        m.created_at
    FROM
        messages m
        CROSS JOIN user_access
    WHERE
        m.channel_id = @channel_id::uuid
        AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
            OR (m.created_at,
                m.id) <(sqlc.narg('cursor_created_at')::timestamptz,
                sqlc.narg('cursor_id')::uuid))
    ORDER BY
        m.created_at DESC,
        m.id DESC
    LIMIT @limit_val::int
)
SELECT
    m.id,
    m.channel_id,
    m.type,
    m.content,
    m.system_metadata,
    m.author_id,
    m.reply_to_message_id,
    m.forwarded_message_id,
    m.forwarded_channel_id,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    target_ids ti
    JOIN messages m ON m.id = ti.id
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
            ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
        FROM
            message_attachments a
        WHERE
            a.message_id = ti.id) att ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
            ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
        FROM
            message_reactions r
        WHERE
            r.message_id = ti.id) rec ON TRUE
ORDER BY
    ti.created_at DESC,
    ti.id DESC;

-- name: MessageListPinnedAggregate :many
WITH hydrated_pinned_messages AS (
    SELECT
        mb.id,
        mb.channel_id,
        mb.reply_to_message_id,
        mb.author_id,
        mb.author_username,
        mb.author_display_name,
        mb.author_avatar_url,
        mb.created_at,
        mb.updated_at,
        mb.edited_at,
        mb.pinned_at,
        mb.content,
        COALESCE(att.attachments, '[]'::json) AS attachments,
        COALESCE(rec.reactions, '[]'::json) AS reactions
    FROM
        message_base_aggregates mb
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
                ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
            FROM
                message_attachments a
            WHERE
                a.message_id = mb.id) att ON TRUE
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
                ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
            FROM
                message_reactions r
            WHERE
                r.message_id = mb.id) rec ON TRUE
        WHERE
            mb.channel_id = @channel_id::uuid
            AND mb.pinned_at IS NOT NULL
            -- Keyset comparison using pinned_at:
            AND (@cursor_pinned_at::timestamptz IS NULL
                OR (mb.pinned_at,
                    mb.id) <(@cursor_pinned_at::timestamptz,
                    @cursor_id::uuid)))
    SELECT
        hm.*
    FROM
        hydrated_pinned_messages hm
    ORDER BY
        hm.pinned_at DESC,
        hm.id DESC
    LIMIT @limit_val::int;

-- name: MessageMemberGet :one
SELECT
    m.id,
    m.channel_id,
    m.author_id,
    m.reply_to_message_id,
    m.forwarded_message_id,
    m.forwarded_channel_id,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    m.type,
    m.content,
    m.system_metadata
FROM
    messages m
WHERE
    m.id = @message_id::uuid
    AND EXISTS (
        SELECT
            1
        FROM
            channel_members cm
        WHERE
            cm.channel_id = m.channel_id
            AND cm.user_id = @user_id::uuid);

-- name: MessageMemberGetFirstUnread :one
SELECT
    m.id,
    m.channel_id,
    m.author_id,
    m.reply_to_message_id,
    m.forwarded_message_id,
    m.forwarded_channel_id,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    m.type,
    m.content,
    m.system_metadata
FROM
    channel_members cm
    JOIN messages m ON m.channel_id = cm.channel_id
        AND m.created_at > COALESCE(cm.last_read_at, to_timestamp(0))
WHERE
    cm.channel_id = @channel_id::uuid
    AND cm.user_id = @user_id::uuid
ORDER BY
    m.created_at ASC,
    m.id ASC
LIMIT 1;

-- name: MessageMemberGetLatest :one
SELECT
    m.id,
    m.channel_id,
    m.author_id,
    m.reply_to_message_id,
    m.forwarded_message_id,
    m.forwarded_channel_id,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    m.type,
    m.content,
    m.system_metadata
FROM
    messages m
WHERE
    m.channel_id = @channel_id::uuid
    AND EXISTS (
        SELECT
            1
        FROM
            channel_members cm
        WHERE
            cm.channel_id = m.channel_id
            AND cm.user_id = @user_id::uuid)
ORDER BY
    m.created_at DESC,
    m.id DESC
LIMIT 1;

-- name: MessageTogglePinned :one
UPDATE
    messages
SET
    pinned_at = CASE WHEN pinned_at IS NULL THEN
        @pinned_at::timestamptz
    ELSE
        NULL
    END,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    *;

-- name: MessageUpdateContent :one
UPDATE
    messages
SET
    content = sqlc.narg('content')::text,
    updated_at = @updated_at::timestamptz,
    edited_at = @edited_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    *;

