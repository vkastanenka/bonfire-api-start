CREATE TABLE user_profiles(
    -- Primary key
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Audit
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- App logic
    display_name varchar(32),
    -- Constraints
    CONSTRAINT display_name_length CHECK (char_length(display_name) >= 3)
);

-- Indexes
CREATE INDEX idx_user_profiles_display_name ON user_profiles(lower(display_name));

-- Triggers
CREATE TRIGGER update_user_profiles_modtime
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

