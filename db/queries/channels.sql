-- name: ChannelCreate :one
INSERT INTO channels(id, created_at, updated_at, type, name, icon_url, peer_id)
    VALUES (@id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('name')::text, sqlc.narg('icon_url')::text, sqlc.narg('peer_id')::uuid)
RETURNING
    id, created_at, updated_at, type, name, icon_url, peer_id;

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
    icon_url,
    peer_id,
    last_message_id,
    last_message_at
FROM
    channels
WHERE
    id = @id::uuid;

-- name: ChannelGetForMember :one
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.type,
    c.name,
    c.icon_url,
    c.peer_id,
    c.last_message_id,
    c.last_message_at
FROM
    channels c
WHERE
    c.id = @channel_id::uuid
    AND EXISTS (
        SELECT
            1
        FROM
            channel_members cm
        WHERE
            cm.channel_id = c.id
            AND cm.user_id = @user_id::uuid);

-- name: ChannelGetForMemberUpdate :one
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.type,
    c.name,
    c.icon_url,
    c.peer_id,
    c.last_message_id,
    c.last_message_at
FROM
    channels c
WHERE
    c.id = @channel_id::uuid
    AND EXISTS (
        SELECT
            1
        FROM
            channel_members cm
        WHERE
            cm.channel_id = c.id
            AND cm.user_id = @user_id::uuid)
FOR UPDATE
    OF c;

-- name: ChannelGetLastMessageId :one
SELECT
    id,
    created_at
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
ORDER BY
    created_at DESC,
    id DESC
LIMIT 1;

-- name: ChannelHasMessagesAfter :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            messages
        WHERE
            channel_id = @channel_id::uuid
            AND (created_at,
                id) >(@created_at::timestamptz,
                @message_id::uuid));

-- name: ChannelHasMessagesBefore :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            messages
        WHERE
            channel_id = @channel_id::uuid
            AND (created_at,
                id) <(@created_at::timestamptz,
                @message_id::uuid));

-- name: ChannelListAggregateByUser :many
WITH user_channels AS (
    SELECT
        cm.channel_id,
        cm.user_id,
        cm.last_read_message_id,
        cm.mention_count,
        cm.created_at AS member_created_at,
        cm.last_read_at,
        cm.pinned_at AS member_pinned_at,
        cm.is_visible,
        c.type AS channel_type,
        c.name AS channel_name,
        c.icon_url AS channel_icon_url,
        c.last_message_id AS channel_last_message_id,
        c.last_message_at AS channel_last_message_at,
        c.peer_id AS peer_user_id,
        c.updated_at AS channel_updated_at
    FROM
        channel_members cm
        JOIN channels c ON c.id = cm.channel_id
    WHERE
        cm.user_id = @user_id::uuid
        AND cm.is_visible = TRUE
        AND (@cursor_channel_id::uuid IS NULL
            OR ((cm.pinned_at IS NOT NULL
                    AND @cursor_pinned_at::timestamptz IS NOT NULL
                    AND (cm.pinned_at,
                        c.updated_at,
                        c.id) <(@cursor_pinned_at::timestamptz,
                        @cursor_updated_at::timestamptz,
                        @cursor_channel_id::uuid))
                OR (cm.pinned_at IS NULL
                    AND @cursor_pinned_at::timestamptz IS NOT NULL)
                OR (cm.pinned_at IS NULL
                    AND @cursor_pinned_at::timestamptz IS NULL
                    AND (c.updated_at,
                        c.id) <(@cursor_updated_at::timestamptz,
                        @cursor_channel_id::uuid))))
    ORDER BY
        cm.pinned_at DESC NULLS LAST,
        c.updated_at DESC,
        c.id DESC
    LIMIT @limit_val::int
)
SELECT
    uc.channel_id,
    uc.user_id,
    uc.channel_type,
    uc.channel_name,
    uc.channel_icon_url,
    uc.peer_user_id,
    uc.channel_last_message_id,
    uc.channel_last_message_at,
    uc.last_read_message_id,
    uc.mention_count,
    uc.member_created_at,
    uc.last_read_at,
    uc.member_pinned_at,
    uc.is_visible,
    uc.channel_updated_at
FROM
    user_channels uc
ORDER BY
    uc.member_pinned_at DESC NULLS LAST,
    uc.channel_updated_at DESC,
    uc.channel_id DESC;

-- name: ChannelUpdate :one
UPDATE
    channels
SET
    name = sqlc.narg('name')::text,
    icon_url = sqlc.narg('icon_url')::text,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    id,
    created_at,
    updated_at,
    type,
    name,
    icon_url,
    peer_id;

