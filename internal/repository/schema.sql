-- 1. Create a reusable function for updating timestamps
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

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

-- user profiles

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