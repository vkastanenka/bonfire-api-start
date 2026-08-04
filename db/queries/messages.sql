-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, reply_to_message_id, author_id, created_at, updated_at, edited_at, pinned_at, content)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('author_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, sqlc.narg('edited_at')::timestamptz, sqlc.narg('pinned_at')::timestamptz, sqlc.narg('content')::text)
RETURNING
    *;

-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = @id::uuid;

-- name: MessageGet :one
SELECT
    id,
    channel_id,
    reply_to_message_id,
    author_id,
    created_at,
    updated_at,
    edited_at,
    pinned_at,
    content
FROM
    messages
WHERE
    id = @id::uuid;

-- name: MessageGetAggregate :one
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
    mb.id = @id::uuid;

-- name: MessageGetFirstUnread :one
-- Fetches the first message created after the user's last_read_at timestamp
SELECT
    m.*
FROM
    messages m
    JOIN channel_members cm ON cm.channel_id = m.channel_id
        AND cm.user_id = @user_id::uuid
WHERE
    m.channel_id = @channel_id::uuid
    AND m.created_at > cm.last_read_at
ORDER BY
    m.created_at ASC,
    m.id ASC
LIMIT 1;

-- name: MessageGetLatest :one
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
ORDER BY
    created_at DESC,
    id DESC
LIMIT 1;

-- name: MessageListAggregateAfter :many
WITH target_ids AS (
    SELECT
        id,
        created_at
    FROM
        messages
    WHERE
        channel_id = @channel_id::uuid
        AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
            OR (created_at,
                id) >(sqlc.narg('cursor_created_at')::timestamptz,
                sqlc.narg('cursor_id')::uuid))
    ORDER BY
        created_at ASC,
        id ASC
    LIMIT @limit_val::int
)
SELECT
    mb.id,
    mb.channel_id,
    mb.type,
    mb.content,
    mb.system_metadata,
    mb.author_id,
    mb.author_username,
    mb.author_display_name,
    mb.author_avatar_url,
    mb.author_banner_color,
    mb.author_disabled_at,
    mb.reply_to_message_id,
    mb.reply_to_author_id,
    mb.reply_to_author_display_name,
    mb.reply_to_author_avatar_url,
    mb.reply_to_content,
    mb.forwarded_message_id,
    mb.forwarded_channel_id,
    mb.forwarded_channel_name,
    mb.created_at,
    mb.updated_at,
    mb.edited_at,
    mb.pinned_at,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    target_ids ti
    JOIN message_base_aggregates mb ON mb.id = ti.id
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
-- Fetches older/target messages and newer messages relative to target, returned ASC
WITH target AS (
    SELECT
        created_at
    FROM
        messages
    WHERE
        id = @target_id::uuid
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
    around_ids ai
    JOIN message_base_aggregates mb ON mb.id = ai.id
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
ORDER BY
    ai.created_at ASC,
    ai.id ASC;

-- name: MessageListAggregateBefore :many
WITH target_ids AS (
    SELECT
        id,
        created_at
    FROM
        messages
    WHERE
        channel_id = @channel_id::uuid
        AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
            OR (created_at,
                id) <(sqlc.narg('cursor_created_at')::timestamptz,
                sqlc.narg('cursor_id')::uuid))
    ORDER BY
        created_at DESC,
        id DESC
    LIMIT @limit_val::int
)
SELECT
    mb.id,
    mb.channel_id,
    mb.type,
    mb.content,
    mb.system_metadata,
    mb.author_id,
    mb.author_username,
    mb.author_display_name,
    mb.author_avatar_url,
    mb.author_banner_color,
    mb.author_disabled_at,
    mb.reply_to_message_id,
    mb.reply_to_author_id,
    mb.reply_to_author_display_name,
    mb.reply_to_author_avatar_url,
    mb.reply_to_content,
    mb.forwarded_message_id,
    mb.forwarded_channel_id,
    mb.forwarded_channel_name,
    mb.created_at,
    mb.updated_at,
    mb.edited_at,
    mb.pinned_at,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    target_ids ti
    JOIN message_base_aggregates mb ON mb.id = ti.id
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

