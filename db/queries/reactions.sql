-- name: MessageReactionAdd :exec
INSERT INTO message_reactions(message_id, user_id, emoji, created_at)
    VALUES (@message_id::uuid, @user_id::uuid, @emoji::text, @created_at::timestamptz)
ON CONFLICT (message_id, user_id, emoji)
    DO NOTHING;

-- name: MessageReactionRemove :exec
DELETE FROM message_reactions
WHERE message_id = @message_id::uuid
    AND user_id = @user_id::uuid
    AND emoji = @emoji::text;

