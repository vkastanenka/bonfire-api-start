-- name: ReactionMemberAdd :one
INSERT INTO message_reactions(message_id, user_id, emoji)
SELECT
    @message_id::uuid,
    @user_id::uuid,
    @emoji::text
WHERE
    EXISTS (
        SELECT
            1
        FROM
            messages m
            JOIN channel_members cm ON cm.channel_id = m.channel_id
        WHERE
            m.id = @message_id::uuid
            AND cm.user_id = @user_id::uuid)
ON CONFLICT (message_id,
    user_id,
    emoji)
    DO UPDATE SET
        emoji = EXCLUDED.emoji
    RETURNING
        message_id,
        user_id,
        emoji,
        created_at;

-- name: ReactionMemberRemove :exec
DELETE FROM message_reactions mr
WHERE mr.message_id = @message_id::uuid
    AND mr.user_id = @user_id::uuid
    AND mr.emoji = @emoji::text
    AND EXISTS (
        SELECT
            1
        FROM
            messages m
            JOIN channel_members cm ON cm.channel_id = m.channel_id
        WHERE
            m.id = mr.message_id
            AND cm.user_id = @user_id::uuid);

