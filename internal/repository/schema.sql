-- 1. Enable the Case-Insensitive Text extension
CREATE EXTENSION IF NOT EXISTS citext;

-- 2. Create a reusable function for updating timestamps
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- outbox_events

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,

    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    last_error TEXT DEFAULT NULL,
    
    attempts INT DEFAULT 0 NOT NULL,
    max_attempts INT DEFAULT 5 NOT NULL,
    next_attempt_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    CONSTRAINT check_attempts_ceiling CHECK (attempts <= max_attempts)
);

CREATE TRIGGER update_outbox_events_modtime
    BEFORE UPDATE ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- sessions

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,

    refresh_token VARCHAR(512) NOT NULL,
    user_agent TEXT NOT NULL,
    client_ip VARCHAR(45) NOT NULL,
    is_blocked BOOLEAN NOT NULL DEFAULT false
);

CREATE TRIGGER update_sessions_modtime
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- users

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,

    verified_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,

    email CITEXT NOT NULL UNIQUE CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    username CITEXT NOT NULL UNIQUE CONSTRAINT username_length CHECK (char_length(username) BETWEEN 8 AND 32),
    password_hash VARCHAR(255) NOT NULL
);

CREATE UNIQUE INDEX users_active_email_idx ON users (email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_active_username_idx ON users (username) WHERE deleted_at IS NULL;

CREATE TRIGGER update_users_modtime
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- user_profiles

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    display_name VARCHAR(32)
);

CREATE TRIGGER update_user_profiles_modtime
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();