CREATE TABLE user_sessions(
    -- Identity
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Audit
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Lifecycle
    expires_at timestamp with time zone NOT NULL,
    last_seen_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Security
    refresh_token varchar(512) NOT NULL UNIQUE,
    is_blocked boolean NOT NULL DEFAULT FALSE,
    -- Client context
    client_ip inet NOT NULL,
    user_agent text NOT NULL
);

-- Indexes
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);

CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Triggers
CREATE TRIGGER update_sessions_modtime
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

