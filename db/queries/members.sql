-- name: ChannelMemberAddBatch :exec
INSERT INTO channel_members(channel_id, user_id, created_at, updated_at, last_read_at, mention_count, last_read_message_id, dm_visibility, pinned_at)
SELECT
    @channel_id::uuid,
    u.user_id,
    u.created_at,
    u.updated_at,
    u.last_read_at,
    u.mention_count,
    u.last_read_message_id,
    u.dm_visibility,
    u.pinned_at
FROM
    ROWS
FROM (unnest(@user_ids::uuid[]),
    unnest(@created_ats::timestamptz[]),
    unnest(@updated_ats::timestamptz[]),
    unnest(@last_read_ats::timestamptz[]),
    unnest(@mention_counts::int[]),
    unnest(@last_read_message_ids::uuid[]),
    unnest(@dm_visibilities::smallint[]),
    unnest(@pinned_ats::timestamptz[])) AS u(user_id,
        created_at,
        updated_at,
        last_read_at,
        mention_count,
        last_read_message_id,
        dm_visibility,
        pinned_at)
ON CONFLICT (channel_id,
    user_id)
    DO UPDATE SET
        updated_at = EXCLUDED.updated_at,
        last_read_at = EXCLUDED.last_read_at,
        mention_count = EXCLUDED.mention_count,
        last_read_message_id = EXCLUDED.last_read_message_id,
        dm_visibility = EXCLUDED.dm_visibility,
        pinned_at = EXCLUDED.pinned_at;

-- name: ChannelMemberCount :one
SELECT
    COUNT(*)::int AS count
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid;

-- name: ChannelMemberGet :one
SELECT
    *
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberGetUnreadCount :one
SELECT
    COUNT(*)::int AS unread_count
FROM
    messages m
    JOIN channel_members cm ON cm.channel_id = m.channel_id
        AND cm.user_id = @user_id::uuid
WHERE
    m.channel_id = @channel_id::uuid
    AND m.created_at > cm.last_read_at;

-- name: ChannelMemberIncrementMentionCountBatch :exec
UPDATE
    channel_members
SET
    mention_count = mention_count + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = ANY (@user_ids::uuid[]);

-- name: ChannelMemberListByChannel :many
SELECT
    channel_id,
    user_id,
    created_at,
    updated_at,
    last_read_at,
    mention_count,
    last_read_message_id,
    pinned_at,
    dm_visibility
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid
ORDER BY
    created_at ASC,
    user_id ASC;

-- name: ChannelMemberListItemsByChannel :many
SELECT
    cm.channel_id,
    cm.user_id,
    cm.created_at AS member_since,
    cm.last_read_at,
    ua.username,
    ua.display_name,
    ua.avatar_url,
    ua.created_at AS user_created_at
FROM
    channel_members cm
    JOIN user_aggregates ua ON ua.id = cm.user_id
WHERE
    cm.channel_id = @channel_id::uuid
    AND (@cursor_created_at::timestamptz IS NULL
        OR (cm.created_at,
            cm.user_id) >(@cursor_created_at::timestamptz,
            @cursor_user_id::uuid))
ORDER BY
    cm.created_at ASC,
    cm.user_id ASC
LIMIT @limit_val::int;

-- name: ChannelMemberRemove :exec
DELETE FROM channel_members
WHERE channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberResetMentionCount :exec
UPDATE
    channel_members
SET
    mention_count = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberTogglePinned :exec
UPDATE
    channel_members
SET
    pinned_at = CASE WHEN pinned_at IS NULL THEN
        @pinned_at::timestamptz
    ELSE
        NULL
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberUpdateRead :exec
UPDATE
    channel_members
SET
    last_read_message_id = CASE WHEN @last_read_at::timestamptz > last_read_at THEN
        COALESCE(sqlc.narg('last_read_message_id')::uuid, last_read_message_id)
    ELSE
        last_read_message_id
    END,
    last_read_at = GREATEST(last_read_at, @last_read_at::timestamptz),
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberCloseDM :exec
-- Hides or closes the DM channel for a specific member
UPDATE
    channel_members
SET
    dm_visibility = 0, -- 0: HIDDEN
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberOpenDM :exec
-- Unhides/reopens the DM channel
UPDATE
    channel_members
SET
    dm_visibility = 1, -- 1: VISIBLE
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- count number of messages in a channel that came after a message id
-- SELECT
--     COUNT(*)::int
-- FROM
--     messages
-- WHERE
--     channel_id = $1
--     AND id > $2; -- last_read_message_id
