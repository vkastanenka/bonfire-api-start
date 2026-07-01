-- messages table
CREATE TABLE messages(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Optimization: This index allows you to load the last N messages
-- for a channel in O(log N) time.
CREATE INDEX idx_messages_channel_created_at ON messages(channel_id, created_at DESC);

