-- name: ChannelMemberCreateBatch :many
INSERT INTO channel_members(channel_id, user_id, last_read_message_id, created_at, updated_at, last_read_message_at, pinned_at, muted_until, mention_count, is_visible)
SELECT
    m.channel_id,
    m.user_id,
    m.last_read_message_id,
    m.created_at,
    m.updated_at,
    m.last_read_message_at,
    m.pinned_at,
    m.muted_until,
    m.mention_count,
    m.is_visible
FROM
    UNNEST(@channel_ids::uuid[], @user_ids::uuid[], @last_read_message_ids::uuid[], @created_ats::timestamptz[], @updated_ats::timestamptz[], @last_read_message_ats::timestamptz[], @pinned_ats::timestamptz[], @muted_untils::timestamptz[], @mention_counts::int[], @is_visibles::boolean[]) AS m(channel_id,
        user_id,
        last_read_message_id,
        created_at,
        updated_at,
        last_read_message_at,
        pinned_at,
        muted_until,
        mention_count,
        is_visible)
ON CONFLICT (channel_id,
    user_id)
    DO UPDATE SET
        last_read_message_id = EXCLUDED.last_read_message_id,
        updated_at = EXCLUDED.updated_at,
        last_read_message_at = EXCLUDED.last_read_message_at,
        pinned_at = EXCLUDED.pinned_at,
        muted_until = EXCLUDED.muted_until,
        mention_count = EXCLUDED.mention_count,
        is_visible = EXCLUDED.is_visible
    RETURNING
        channel_members.*;

-- name: ChannelMemberGet :one
SELECT
    channel_members.*
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberGetBatchByChannelIDs :many
SELECT
    channel_members.*
FROM
    channel_members
WHERE
    channel_id = ANY (@channel_ids::uuid[]);

-- name: ChannelMemberListVisibleByUserID :many
SELECT
    channel_members.*
FROM
    channel_members
WHERE
    user_id = @user_id::uuid
    AND is_visible = TRUE
ORDER BY
    (pinned_at IS NOT NULL) DESC,
    pinned_at DESC NULLS LAST,
    id DESC
LIMIT @limit_val::int;

-- name: ChannelMemberUpdateLastReadMessage :one
UPDATE
    channel_members
SET
    last_read_message_id = @last_read_message_id::uuid,
    last_read_message_at = @last_read_message_at::timestamptz,
    mention_count = COALESCE(sqlc.narg('mention_count')::int, mention_count),
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid
RETURNING
    channel_members.*;

-- name: ChannelMemberUpdatePinnedAt :one
UPDATE
    channel_members
SET
    pinned_at = sqlc.narg('pinned_at')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid
RETURNING
    channel_members.*;

-- name: ChannelMemberUpdateMutedUntil :one
UPDATE
    channel_members
SET
    muted_until = sqlc.narg('muted_until')::timestamptz,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid
RETURNING
    channel_members.*;

-- name: ChannelMemberUpdateIsVisible :one
UPDATE
    channel_members
SET
    is_visible = @is_visible::boolean,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid
RETURNING
    channel_members.*;

-- name: ChannelMemberIncrementBatchMentionCount :exec
UPDATE
    channel_members
SET
    mention_count = mention_count + 1,
    is_visible = TRUE,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = ANY (@user_ids::uuid[])
    AND (muted_until IS NULL
        OR muted_until < CURRENT_TIMESTAMP);

-- name: ChannelMemberDelete :exec
DELETE FROM channel_members
WHERE channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

