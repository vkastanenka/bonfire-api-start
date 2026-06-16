--------
-- users
CREATE TABLE users (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    -- Audit metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Core identity
    email CITEXT NOT NULL UNIQUE,
    username CITEXT NOT NULL UNIQUE,
    -- Auth / security
    password_hash VARCHAR(255) NOT NULL,
    is_totp_enabled BOOLEAN DEFAULT FALSE NOT NULL,
    totp_secret VARCHAR(255),
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    last_verification_sent_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    -- App logic
    flags BIGINT NOT NULL DEFAULT 0,
    -- Constraints
    CONSTRAINT email_length CHECK (
        char_length(email) BETWEEN 3
        AND 255
    ),
    CONSTRAINT username_length CHECK (
        char_length(username) BETWEEN 8
        AND 32
    ),
    CONSTRAINT username_reserved CHECK (
        lower(username) NOT IN (
            'admin',
            'root',
            'support',
            'system',
            'moderator'
        )
    );

);

-- Comments
COMMENT ON COLUMN users.flags IS 'Bitmask: 1=admin, 2=user';

-- Indexes
CREATE INDEX idx_users_unverified_cleanup ON users(created_at)
WHERE
    verified_at IS NULL;

CREATE INDEX idx_users_flags ON users(flags);

-- Triggers
CREATE TRIGGER update_users_modtime BEFORE
UPDATE
    ON users FOR EACH ROW EXECUTE FUNCTION update_modified_column();

-------------------------
-- user_deletion_requests
CREATE TABLE user_deletion_requests (
    -- Primary key
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Audit metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    -- Constraints
    CONSTRAINT valid_schedule_date CHECK (scheduled_at > created_at)
);

-- Indexes
CREATE INDEX idx_deletion_requests_scheduled_at ON user_deletion_requests(scheduled_at);

----------------
-- user_profiles
CREATE TABLE user_profiles (
    -- Primary key
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- App logic
    display_name VARCHAR(32),
    -- Constraints
    CONSTRAINT display_name_length CHECK (char_length(display_name) >= 3)
);

-- Indexes
CREATE INDEX idx_user_profiles_display_name ON user_profiles(lower(display_name));

-- Triggers
CREATE TRIGGER update_user_profiles_modtime BEFORE
UPDATE
    ON user_profiles FOR EACH ROW EXECUTE FUNCTION update_modified_column();