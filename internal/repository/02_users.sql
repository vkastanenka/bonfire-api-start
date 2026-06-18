-- Types
CREATE TYPE user_role AS ENUM(
    'user',
    'admin'
);

CREATE TABLE users(
    -- Primary key
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- Audit metadata
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Core identity
    email CITEXT NOT NULL UNIQUE,
    username CITEXT NOT NULL UNIQUE,
    -- Auth / security
    password_hash varchar(255) NOT NULL,
    is_totp_enabled boolean DEFAULT FALSE NOT NULL,
    totp_secret varchar(255),
    verified_at timestamp with time zone DEFAULT NULL,
    last_verification_sent_at timestamp with time zone DEFAULT NULL,
    -- App logic
    -- ROLE user_role NOT NULL DEFAULT 'user',
    -- Constraints
    CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 8 AND 32),
    CONSTRAINT username_reserved CHECK (lower(username) NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire'))
);

-- Indexes
CREATE INDEX idx_users_unverified ON users(created_at)
WHERE
    verified_at IS NULL;

CREATE INDEX idx_users_role ON users(ROLE);

-- Triggers
CREATE TRIGGER update_users_modtime
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

