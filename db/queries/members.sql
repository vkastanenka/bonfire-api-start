-- name: ChannelMemberCreateBatch :many
INSERT INTO channel_members(channel_id, user_id, last_read_message_id, created_at, updated_at, last_read_message_at, pinned_at, muted_until, mention_count, is_visible)
SELECT
    channel_id,
    user_id,
    last_read_message_id,
    created_at,
    updated_at,
    last_read_message_at,
    pinned_at,
    muted_until,
    mention_count,
    is_visible
FROM
    jsonb_to_recordset(@payload::jsonb) AS x(channel_id uuid,
        user_id uuid,
        last_read_message_id uuid,
        created_at timestamptz,
        updated_at timestamptz,
        last_read_message_at timestamptz,
        pinned_at timestamptz,
        muted_until timestamptz,
        mention_count integer,
        is_visible boolean)
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
    pinned_at DESC NULLS LAST,
    channel_id DESC
LIMIT @limit_val::int;

-- name: ChannelMemberCountByChannelID :one
SELECT
    COUNT(*)::bigint
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid;

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

-- name: ChannelMemberUpdateLastReadMessage :one
UPDATE
    channel_members
SET
    last_read_message_id = sqlc.narg('last_read_message_id')::uuid,
    last_read_message_at = @last_read_message_at::timestamptz,
    mention_count = CASE WHEN sqlc.narg('mention_count')::int IS NOT NULL THEN
        sqlc.narg('mention_count')::int
    ELSE
        mention_count
    END,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid
    AND @last_read_message_at::timestamptz >= last_read_message_at
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

-- name: ChannelMemberIncrementPeersMentionCountByChannelID :exec
UPDATE
    channel_members
SET
    mention_count = mention_count + @increment_amount::integer,
    is_visible = TRUE,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id != @user_id::uuid
    AND (muted_until IS NULL
        OR muted_until < @updated_at::timestamptz);

-- name: ChannelMemberDelete :exec
DELETE FROM channel_members
WHERE channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberClearBatchLastReadMessageByChannelID :many
UPDATE
    channel_members
SET
    last_read_message_id = NULL,
    last_read_message_at = @last_read_message_at::timestamptz,
    mention_count = 0,
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND last_read_message_id IS NOT NULL
RETURNING
    channel_members.*;

