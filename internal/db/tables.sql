CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION update_modified_column()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TABLE outbox_events(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    locked_by uuid,
    processed_at timestamptz,
    next_attempt_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lease_expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    attempts int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 5,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    last_error text,
    CONSTRAINT event_type_length CHECK (length(event_type) BETWEEN 1 AND 100),
    CONSTRAINT payload_populated CHECK (payload != '{}'::jsonb AND payload != '[]'::jsonb),
    CONSTRAINT payload_size_limit CHECK (pg_column_size(payload) < 102400),
    CONSTRAINT attempts_limit CHECK (attempts <= max_attempts),
    CONSTRAINT last_error_length CHECK (last_error IS NULL OR length(last_error) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_outbox_events_unprocessed ON outbox_events(next_attempt_at ASC, id ASC)
WHERE
    processed_at IS NULL;

CREATE TRIGGER update_outbox_events_modtime
    BEFORE UPDATE ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE users(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    verified_at timestamp with time zone DEFAULT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    preferred_presence smallint DEFAULT NULL,
    email CITEXT NOT NULL UNIQUE,
    username CITEXT NOT NULL UNIQUE,
    password_hash text NOT NULL,
    CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 3 AND 32),
    CONSTRAINT username_reserved CHECK (lower(username) NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire')),
    CONSTRAINT password_hash_length CHECK (char_length(password_hash) BETWEEN 3 AND 255),
    CONSTRAINT preferred_presence_values CHECK (preferred_presence IN (4, 5, 6))
);

CREATE TRIGGER update_users_modtime
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE user_profiles(
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    display_name text NOT NULL,
    avatar_url text,
    CONSTRAINT display_name_length CHECK (char_length(display_name) BETWEEN 3 AND 32),
    CONSTRAINT avatar_url_length CHECK (char_length(avatar_url) BETWEEN 3 AND 2048)
);

CREATE INDEX idx_user_profiles_display_name ON user_profiles(display_name);

CREATE TRIGGER update_user_profiles_modtime
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE sessions(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_seen_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    client_ip inet NOT NULL,
    refresh_token_hash bytea NOT NULL UNIQUE,
    user_agent text NOT NULL,
    os text NOT NULL DEFAULT 'Unknown',
    browser text NOT NULL DEFAULT 'Unknown',
    CONSTRAINT refresh_token_hash_length CHECK (octet_length(refresh_token_hash) = 32),
    CONSTRAINT user_agent_length CHECK (length(user_agent) BETWEEN 1 AND 1000),
    CONSTRAINT os_length CHECK (length(os) BETWEEN 1 AND 100),
    CONSTRAINT browser_length CHECK (length(browser) BETWEEN 1 AND 100)
);

CREATE INDEX idx_sessions_user_active ON sessions(user_id)
WHERE (revoked_at IS NULL);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TRIGGER update_sessions_modtime
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE channels(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    type SMALLINT NOT NULL,
    name text,
    CONSTRAINT valid_channel_type CHECK (type IN (0, 1, 2, 3, 4)),
    CONSTRAINT check_channel_name_rules CHECK ((type = 1 AND name IS NULL) OR (type != 1 AND (name IS NULL OR length(trim(name)) BETWEEN 1 AND 100)))
);

CREATE TRIGGER update_channels_modtime
    BEFORE UPDATE ON channels
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE channel_members(
    channel_id uuid REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_user_id ON channel_members(user_id);

CREATE TRIGGER update_channel_members_modtime
    BEFORE UPDATE ON channel_members
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE TABLE relationships(
    user1_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    type SMALLINT NOT NULL, -- (1 = pending, 2 = friends, 3 = blocked)
    PRIMARY KEY (user1_id, user2_id),
    -- CRITICAL: Enforce alphabetical order to ensure A->B and B->A are the same row
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT valid_relationship_type CHECK (type IN (1, 2, 3))
);

CREATE INDEX idx_relationships_user2_type ON relationships(user2_id, type);

CREATE TRIGGER update_relationships_modtime
    BEFORE UPDATE ON relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

CREATE OR REPLACE VIEW relationship_perspectives AS
-- ==========================================
-- PERSPECTIVE A: The querying user is user1_id, looking at user2_id (the peer)
-- ==========================================
SELECT
    r.user1_id AS user_id,
    r.user2_id AS peer_id,
    r.type,
    r.actor_id,
(r.actor_id = r.user1_id) AS is_initiator, -- TRUE = Sent by user1, FALSE = Received by user1
    r.created_at,
    r.updated_at,
    u2.username,
    p2.display_name,
    p2.avatar_url,
    u2.preferred_presence AS user_preferred_presence, -- Matches 'preferred_presence' column in users table
(
        SELECT
            cm1.channel_id
        FROM channel_members cm1
        JOIN channel_members cm2 ON cm1.channel_id = cm2.channel_id
        JOIN channels c ON cm1.channel_id = c.id
        WHERE
            c.type = 1 -- 1 = DM Channel
            AND cm1.user_id = r.user1_id
            AND cm2.user_id = r.user2_id LIMIT 1) AS channel_id
FROM
    relationships r
    JOIN users u2 ON r.user2_id = u2.id
    LEFT JOIN user_profiles p2 ON r.user2_id = p2.user_id -- Matches 'user_profiles'
WHERE
-- SECURITY: If blocked (type = 3), ONLY show this row to the blocker (user1_id)
(r.type != 3
    OR r.actor_id = r.user1_id)
UNION ALL
-- ==========================================
-- PERSPECTIVE B: The querying user is user2_id, looking at user1_id (the peer)
-- ==========================================
SELECT
    r.user2_id AS user_id,
    r.user1_id AS peer_id,
    r.type,
    r.actor_id,
(r.actor_id = r.user2_id) AS is_initiator, -- TRUE = Sent by user2, FALSE = Received by user2
    r.created_at,
    r.updated_at,
    u1.username,
    p1.display_name,
    p1.avatar_url,
    u1.preferred_presence AS user_preferred_presence, -- Matches 'preferred_presence' column in users table
(
        SELECT
            cm1.channel_id
        FROM channel_members cm1
        JOIN channel_members cm2 ON cm1.channel_id = cm2.channel_id
        JOIN channels c ON cm1.channel_id = c.id
        WHERE
            c.type = 1 -- 1 = DM Channel
            AND cm1.user_id = r.user1_id
            AND cm2.user_id = r.user2_id LIMIT 1) AS channel_id
FROM
    relationships r
    JOIN users u1 ON r.user1_id = u1.id
    LEFT JOIN user_profiles p1 ON r.user1_id = p1.user_id -- Matches 'user_profiles'
WHERE
-- SECURITY: If blocked (type = 3), ONLY show this row to the blocker (user2_id)
(r.type != 3
    OR r.actor_id = r.user2_id);

