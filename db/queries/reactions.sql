-- name: ReactionCreate :one
INSERT INTO message_reactions(message_id, user_id, emoji, created_at)
    VALUES (@message_id::uuid, @user_id::uuid, @emoji::text, @created_at::timestamptz)
RETURNING
    message_reactions.*;

-- name: ReactionGet :one
SELECT
    message_reactions.*
FROM
    message_reactions
WHERE
    message_id = @message_id::uuid
    AND user_id = @user_id::uuid
    AND emoji = @emoji::text;

-- name: ReactionGetBatchByUserIDAndMessageIDs :many
SELECT
    message_id,
    emoji
FROM
    message_reactions
WHERE
    message_id = ANY (@message_ids::uuid[])
    AND user_id = @user_id::uuid;

-- name: ReactionGetBatchSummaryByMessageIDs :many
SELECT
    message_id,
    emoji,
    COUNT(*)::int AS count
FROM
    message_reactions
WHERE
    message_id = ANY (@message_ids::uuid[])
GROUP BY
    message_id,
    emoji;

-- name: ReactionCountByEmoji :one
SELECT
    COUNT(*)::bigint
FROM
    message_reactions
WHERE
    message_id = @message_id::uuid
    AND emoji = @emoji::text;

-- name: ReactionDelete :exec
DELETE FROM message_reactions
WHERE message_id = @message_id::uuid
    AND user_id = @user_id::uuid
    AND emoji = @emoji::text;

