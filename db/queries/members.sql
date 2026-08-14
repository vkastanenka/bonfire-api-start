-- -- name: ChannelMemberAddBatch :exec
-- WITH input_data AS (
--     SELECT
--         *
--     FROM
--         jsonb_populate_recordset(NULL::channel_members, @members_json::jsonb))
-- INSERT INTO channel_members(channel_id, user_id, created_at, updated_at, last_read_at, mention_count, last_read_message_id, is_visible, pinned_at)
-- SELECT
--     channel_id,
--     user_id,
--     created_at,
--     updated_at,
--     last_read_at,
--     mention_count,
--     last_read_message_id,
--     is_visible,
--     pinned_at
-- FROM
--     input_data
-- ON CONFLICT (channel_id,
--     user_id)
--     DO UPDATE SET
--         updated_at = EXCLUDED.updated_at,
--         last_read_at = EXCLUDED.last_read_at,
--         mention_count = EXCLUDED.mention_count,
--         last_read_message_id = EXCLUDED.last_read_message_id,
--         is_visible = EXCLUDED.is_visible,
--         pinned_at = EXCLUDED.pinned_at;
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
    created_at DESC
LIMIT @limit_val::int;

-- -- name: ChannelMemberTogglePinned :exec
-- UPDATE
--     channel_members
-- SET
--     pinned_at = CASE WHEN pinned_at IS NULL THEN
--         @pinned_at::timestamptz
--     ELSE
--         NULL
--     END,
--     updated_at = CURRENT_TIMESTAMP
-- WHERE
--     channel_id = @channel_id::uuid
--     AND user_id = @user_id::uuid;
-- -- name: ChannelMemberUpdateLastRead :exec
-- UPDATE
--     channel_members
-- SET
--     last_read_message_id = CASE WHEN @last_read_at::timestamptz > last_read_at THEN
--         COALESCE(sqlc.narg('last_read_message_id')::uuid, last_read_message_id)
--     ELSE
--         last_read_message_id
--     END,
--     last_read_at = GREATEST(last_read_at, @last_read_at::timestamptz),
--     updated_at = CURRENT_TIMESTAMP
-- WHERE
--     channel_id = @channel_id::uuid
--     AND user_id = @user_id::uuid;
-- -- name: ChannelMemberCloseDM :exec
-- UPDATE
--     channel_members
-- SET
--     is_visible = FALSE,
--     updated_at = @updated_at::timestamptz
-- WHERE
--     channel_id = @channel_id::uuid
--     AND user_id = @user_id::uuid;
-- -- name: ChannelMemberOpenDM :exec
-- UPDATE
--     channel_members
-- SET
--     is_visible = TRUE,
--     updated_at = @updated_at::timestamptz
-- WHERE
--     channel_id = @channel_id::uuid
--     AND user_id = @user_id::uuid;
