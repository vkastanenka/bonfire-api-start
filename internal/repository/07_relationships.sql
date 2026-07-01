-- Table
CREATE TABLE relationships(
    user1_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type SMALLINT NOT NULL, -- Refactored from status enum (0 = pending, 1 = friends, 2 = blocked)
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user1_id, user2_id),
    -- CRITICAL: Enforce alphabetical order to ensure A->B and B->A are the same row
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT check_valid_relationship_type CHECK (type IN (0, 1, 2))
);

CREATE INDEX idx_relationships_user2_type ON relationships(user2_id, type);

CREATE TRIGGER update_relationships_modtime
    BEFORE UPDATE ON relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- =========================================================================
-- PERSPECTIVE VIEW
-- Unrolls the single-row alphabetical constraints into a clean directional mapping
-- =========================================================================
CREATE OR REPLACE VIEW relationship_perspectives AS
-- Perspective A: The querying user is in the user1_id slot (Target user is user2)
SELECT
    r.user1_id AS user_id,
    r.user2_id AS peer_id,
    r.type,
    r.actor_id,
    r.created_at,
    r.updated_at,
    u2.username,
    p2.display_name,
    p2.avatar_url,
    u2.status AS user_status,
(
        SELECT
            cm1.channel_id
        FROM
            channel_members cm1
            JOIN channel_members cm2 ON cm1.channel_id = cm2.channel_id
            JOIN channels c ON cm1.channel_id = c.id
        WHERE
            c.type = 1 -- 1 = DM Channel
            AND cm1.user_id = r.user1_id
            AND cm2.user_id = r.user2_id
        LIMIT 1) AS channel_id
FROM
    relationships r
    JOIN users u2 ON r.user2_id = u2.id
    JOIN profiles p2 ON r.user2_id = p2.user_id
UNION ALL
-- Perspective B: The querying user is in the user2_id slot (Target user is user1)
SELECT
    r.user2_id AS user_id,
    r.user1_id AS peer_id,
    r.type,
    r.actor_id,
    r.created_at,
    r.updated_at,
    u1.username,
    p1.display_name,
    p1.avatar_url,
    u1.status AS user_status,
(
        SELECT
            cm1.channel_id
        FROM
            channel_members cm1
            JOIN channel_members cm2 ON cm1.channel_id = cm2.channel_id
            JOIN channels c ON cm1.channel_id = c.id
        WHERE
            c.type = 1 -- 1 = DM Channel
            AND cm1.user_id = r.user1_id
            AND cm2.user_id = r.user2_id
        LIMIT 1) AS channel_id
FROM
    relationships r
    JOIN users u1 ON r.user1_id = u1.id
    JOIN profiles p1 ON r.user1_id = p1.user_id;

