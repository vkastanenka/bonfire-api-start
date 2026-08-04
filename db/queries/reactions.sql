-- name: ReactionAdd :one
INSERT INTO message_reactions(message_id, user_id, emoji)
    VALUES (@message_id, @user_id, @emoji)
ON CONFLICT (message_id, user_id, emoji)
    DO UPDATE SET
        emoji = EXCLUDED.emoji
    RETURNING
        *;

-- name: ReactionRemove :exec
DELETE FROM message_reactions
WHERE message_id = @message_id
    AND user_id = @user_id
    AND emoji = @emoji;

