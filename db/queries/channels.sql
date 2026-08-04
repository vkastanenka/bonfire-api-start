-- name: ChannelCreate :one
INSERT INTO channels(id, created_at, updated_at, type, last_message_id, name, icon_url)
    VALUES (@id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('last_message_id')::uuid, sqlc.narg('name')::text, sqlc.narg('icon_url')::text)
RETURNING
    *;

-- name: ChannelDelete :exec
DELETE FROM channels
WHERE id = @id::uuid;

-- name: ChannelGet :one
SELECT
    *
FROM
    channels
WHERE
    id = @id::uuid;

-- name: ChannelGetForMember :one
SELECT
    c.*
FROM
    channels c
    INNER JOIN channel_members cm ON cm.channel_id = c.id
WHERE
    c.id = @channel_id::uuid
    AND cm.user_id = @user_id::uuid;

-- name: ChannelGetForMemberUpdate :one
SELECT
    c.*
FROM
    channels c
    INNER JOIN channel_members cm ON cm.channel_id = c.id
WHERE
    c.id = @channel_id::uuid
    AND cm.user_id = @user_id::uuid
FOR UPDATE
    OF c;

-- name: ChannelHasMessagesAfter :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            messages
        WHERE
            channel_id = @channel_id::uuid
            AND (created_at > @created_at::timestamptz
                OR (created_at = @created_at::timestamptz
                    AND id > @message_id::uuid)));

-- name: ChannelHasMessagesBefore :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            messages
        WHERE
            channel_id = @channel_id::uuid
            AND (created_at < @created_at::timestamptz
                OR (created_at = @created_at::timestamptz
                    AND id < @message_id::uuid)));

-- name: ChannelListByUser :many
-- Used for populating the user's sidebar with channel info & peer profiles for DMs
SELECT
    cm.channel_id,
    cm.user_id,
    rp.peer_id AS peer_user_id,
    c.type AS channel_type,
    COALESCE(c.name, rp.display_name, rp.username) AS channel_name,
    COALESCE(c.icon_url, rp.avatar_url) AS channel_icon_url,
    c.last_message_id AS channel_last_message_id,
    cm.last_read_message_id,
    cm.mention_count,
    cm.created_at,
    cm.last_read_at,
    cm.pinned_at AS member_pinned_at,
    cm.dm_visibility,
    c.updated_at AS channel_updated_at
FROM
    channel_members cm
    JOIN channels c ON cm.channel_id = c.id
    LEFT JOIN relationship_perspectives rp ON c.type = 0
        AND rp.user_id = cm.user_id
        AND rp.channel_id = c.id
WHERE
    cm.user_id = @user_id::uuid
    AND cm.dm_visibility = 1 -- 1: VISIBLE
ORDER BY
    c.updated_at DESC,
    c.id ASC;

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
    *;

-- name: ChannelUpdateLastMessage :one
UPDATE
    channels
SET
    last_message_id = @last_message_id::uuid,
    updated_at = @updated_at::timestamptz
WHERE
    id = @channel_id::uuid
    AND updated_at <= @updated_at::timestamptz
RETURNING
    *;

