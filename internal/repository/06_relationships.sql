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
    action_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status relationship_status NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user1_id, user2_id),
    -- CRITICAL: Enforce alphabetical order to ensure A->B and B->A are the same row
    CONSTRAINT user_order CHECK (user1_id < user2_id)
);

CREATE INDEX idx_relationships_user2_status ON relationships(user2_id, status);

CREATE TRIGGER update_relationships_modtime
    BEFORE UPDATE ON relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

