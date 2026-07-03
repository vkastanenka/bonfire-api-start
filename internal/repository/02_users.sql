CREATE TABLE users(
    -- Primary key
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- Core
    email CITEXT NOT NULL UNIQUE,
    username CITEXT NOT NULL UNIQUE,
    -- Auth
    verified_at timestamp with time zone DEFAULT NULL,
    security_version int NOT NULL DEFAULT 0,
    -- Metadata
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Constraints
    CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 8 AND 32),
    CONSTRAINT username_reserved CHECK (lower(username) NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire'))
);

CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_users_username ON users(username);

CREATE TRIGGER update_users_modtime
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- CREATE TABLE user_standings(
--     id varchar(32) PRIMARY KEY,
--     description text NOT NULL
-- );
-- INSERT INTO user_standings(id, description)
-- VALUES
--     ('active', 'User has full application access'),
-- ('suspended', 'Account locked due to security/moderation enforcement'),
-- ('deactivated', 'Account self-terminated by the user');
-- -- Types
-- CREATE TYPE user_status AS ENUM(
--     'active',
--     'suspended'
-- );
-- CREATE TYPE user_role AS ENUM(
--     'user',
--     'admin'
-- );
-- CREATE TYPE user_presence AS ENUM(
--     'default',
--     'busy',
--     'dnd'
--     'invisible'
-- );
-- CREATE TABLE users(
--     -- Primary key
--     id uuid PRIMARY KEY DEFAULT uuidv7(),
--     -- Audit metadata
--     created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
--     updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
--     -- Core identity
--     email CITEXT NOT NULL UNIQUE,
--     username CITEXT NOT NULL UNIQUE,
--     -- Auth / security
--     password_hash varchar(255) NOT NULL,
--     is_totp_enabled boolean DEFAULT FALSE NOT NULL,
--     totp_secret varchar(255),
--     verified_at timestamp with time zone DEFAULT NULL,
--     last_verification_sent_at timestamp with time zone DEFAULT NULL,
--     security_version int NOT NULL DEFAULT 0,
--     -- App logic
--     role user_role NOT NULL DEFAULT 'user',
--     status user_status NOT NULL DEFAULT 'active',
--     presence user_presence NOT NULL DEFAULT 'default',
--     -- Constraints
--     CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
--     CONSTRAINT username_length CHECK (char_length(username) BETWEEN 8 AND 32),
--     CONSTRAINT username_reserved CHECK (lower(username) NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire'))
-- );
-- -- Indexes
-- CREATE INDEX idx_users_unverified ON users(created_at)
-- WHERE
--     verified_at IS NULL;
-- CREATE INDEX idx_users_role ON users(role);
-- -- Triggers
-- CREATE TRIGGER update_users_modtime
--     BEFORE UPDATE ON users
--     FOR EACH ROW
--     EXECUTE FUNCTION update_modified_column();
