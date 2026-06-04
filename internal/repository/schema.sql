-- 1. Create a reusable function for updating timestamps
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- users

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

-- friends

-- 1. Create the custom ENUM state type
CREATE TYPE friend_status AS ENUM ('pending', 'accepted', 'blocked');

-- 2. Create the friends junction table
CREATE TABLE friends (
    -- 'user_id' represents the subject, 'friend_id' represents the object
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status friend_status NOT NULL DEFAULT 'pending',
    
    -- Track who initiated the last action
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Composite primary key
    PRIMARY KEY (user_id, friend_id),
    
    -- Enforce alphabetical/numerical sorting to prevent duplicate inverse pairs
    CONSTRAINT check_user_order CHECK (user_id < friend_id)
);

-- 3. Fixed: Table name updated to 'friends' to match the table above
CREATE TRIGGER update_friends_modtime
    BEFORE UPDATE ON friends
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- 4. Fixed: Index name updated to reflect the 'friends' table
CREATE INDEX idx_friends_friend_id ON friends(friend_id);