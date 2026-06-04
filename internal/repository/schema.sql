-- 1. Create a reusable function for updating timestamps
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Create the users table using native PG18 UUIDv7
CREATE TABLE users (
    -- Secure, time-sortable, unpredictable ID
    id UUID PRIMARY KEY DEFAULT uuidv7(),

    -- Discord specific: username and the optional global display name
    username VARCHAR(32) NOT NULL UNIQUE,
    display_name VARCHAR(32),

    -- Essential auth and communication
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(512),

    -- Timestamps for account age and soft-deletion tracking
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- 3. Attach the trigger to the users table
CREATE TRIGGER update_users_modtime
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();