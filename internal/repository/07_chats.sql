-- Represents the "DM Tab" or "Channel"
CREATE TABLE chats(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    type VARCHAR(20) NOT NULL, -- 'dm', 'group', 'server_channel'
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Join table for participants
CREATE TABLE chat_members(
    conversation_id uuid REFERENCES chats(id) ON DELETE CASCADE,
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (conversation_id, user_id)
);

-- The individual text messages
CREATE TABLE messages(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    conversation_id uuid REFERENCES chats(id) ON DELETE CASCADE,
    author_id uuid REFERENCES users(id) ON DELETE CASCADE,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);

