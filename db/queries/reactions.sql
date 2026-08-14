-- name: ReactionCreate :one
INSERT INTO message_reactions(message_id, user_id, emoji, created_at)
    VALUES (@message_id::uuid, @user_id::uuid, @emoji::text, @created_at::timestamptz)
RETURNING
    message_reactions.*;

-- name: ReactionGetBatchByMessageIDs :many
SELECT
    *
FROM
    message_reactions
WHERE
    message_id = ANY (@message_ids::uuid[])
ORDER BY
    message_id,
    created_at ASC;

-- name: ReactionDelete :exec
DELETE FROM message_reactions
WHERE message_id = @message_id::uuid
    AND user_id = @user_id::uuid
    AND emoji = @emoji::text;

