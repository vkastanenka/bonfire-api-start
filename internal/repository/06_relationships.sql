-- Types
CREATE TYPE relationship_status AS ENUM(
    'pending',
    'friends',
    'blocked'
);

-- Table
CREATE TABLE relationships(
    user1_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status relationship_status NOT NULL,
    action_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Primary Key naturally prevents duplicate relationships between two people
    PRIMARY KEY (user1_id, user2_id),
    -- CRITICAL: Enforce alphabetical order to ensure A->B and B->A are the same row
    CONSTRAINT user_order CHECK (user1_id < user2_id)
);

-- Indexes to quickly look up a user's relationships regardless of column order
CREATE INDEX idx_relationships_user1 ON relationships(user1_id);

CREATE INDEX idx_relationships_user2 ON relationships(user2_id);

CREATE INDEX idx_relationships_status ON relationships(status);

-- Use your existing trigger to keep updated_at accurate
CREATE TRIGGER update_relationships_modtime
    BEFORE UPDATE ON relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

