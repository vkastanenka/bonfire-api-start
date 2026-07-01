-- ==========================================
-- CHAT MEMBERS
-- ==========================================
-- name: IsUserInConversation :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            chat_members
        WHERE
            conversation_id = $1
            AND user_id = $2);

-- ==========================================
-- MESSAGES
-- ==========================================
-- name: MessageCreate :one
INSERT INTO messages(conversation_id, author_id, content)
    VALUES ($1, $2, $3)
RETURNING
    *;

-- name: MessageListByConversation :many
SELECT
    *
FROM
    messages
WHERE
    conversation_id = $1
ORDER BY
    created_at ASC;

