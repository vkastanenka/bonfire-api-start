-- sessions
CREATE TABLE sessions (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Lifecycle
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Security
    refresh_token VARCHAR(512) NOT NULL,
    is_blocked BOOLEAN NOT NULL DEFAULT false,
    -- Client context
    client_ip INET NOT NULL,
    user_agent TEXT NOT NULL
);

-- Indexes
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE UNIQUE INDEX idx_sessions_refresh_token ON sessions(refresh_token);

-- Triggers
CREATE TRIGGER update_sessions_modtime BEFORE
UPDATE
    ON sessions FOR EACH ROW EXECUTE FUNCTION update_modified_column();