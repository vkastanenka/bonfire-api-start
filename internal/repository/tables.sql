CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION update_modified_column()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TABLE users(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    email CITEXT NOT NULL UNIQUE,
    username CITEXT NOT NULL UNIQUE,
    password_hash text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 3 AND 32),
    CONSTRAINT username_reserved CHECK (lower(username) NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire')),
    CONSTRAINT password_hash_length CHECK (char_length(password_hash) BETWEEN 3 AND 255)
);

CREATE TRIGGER update_users_modtime
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE user_profiles(
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name varchar(32) NOT NULL,
    avatar_url varchar(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT display_name_length CHECK (char_length(display_name) BETWEEN 3 AND 32)
);

CREATE INDEX idx_user_profiles_display_name ON user_profiles(display_name);

CREATE TRIGGER update_user_profiles_modtime
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE sessions(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash bytea NOT NULL UNIQUE,
    last_seen_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    client_ip inet NOT NULL,
    user_agent text NOT NULL,
    os text NOT NULL DEFAULT 'Unknown',
    browser text NOT NULL DEFAULT 'Unknown',
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT refresh_token_hash_length CHECK (octet_length(refresh_token_hash) = 32),
    CONSTRAINT user_agent_length CHECK (length(user_agent) BETWEEN 1 AND 1000),
    CONSTRAINT os_length CHECK (length(os) BETWEEN 1 AND 100),
    CONSTRAINT browser_length CHECK (length(browser) BETWEEN 1 AND 100)
);

CREATE INDEX idx_sessions_user_active ON sessions(user_id)
WHERE (revoked_at IS NULL);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TRIGGER update_sessions_modtime
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE outbox_events(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    processed_at timestamptz,
    attempts int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 5,
    next_attempt_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_by uuid,
    lease_expires_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT event_type_length CHECK (length(event_type) BETWEEN 1 AND 100),
    CONSTRAINT payload_populated CHECK (payload != '{}'::jsonb AND payload != '[]'::jsonb),
    CONSTRAINT payload_size_limit CHECK (pg_column_size(payload) < 102400),
    CONSTRAINT attempts_limit CHECK (attempts <= max_attempts),
    CONSTRAINT last_error_length CHECK (last_error IS NULL OR length(last_error) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_outbox_events_unprocessed ON outbox_events(next_attempt_at ASC, id ASC)
WHERE
    processed_at IS NULL AND (lease_expires_at IS NULL OR lease_expires_at < CURRENT_TIMESTAMP);

CREATE TRIGGER update_outbox_events_modtime
    BEFORE UPDATE ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

