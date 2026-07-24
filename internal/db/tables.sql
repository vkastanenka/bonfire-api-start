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
    locked_by uuid DEFAULT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_attempt_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lease_expires_at timestamp with time zone DEFAULT NULL,
    processed_at timestamp with time zone DEFAULT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 5,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    last_error text DEFAULT NULL,
    CONSTRAINT event_type_length CHECK (char_length(event_type) BETWEEN 1 AND 100),
    CONSTRAINT payload_populated CHECK (payload != '{}'::jsonb AND payload != '[]'::jsonb),
    CONSTRAINT payload_size_limit CHECK (pg_column_size(payload) < 102400),
    CONSTRAINT attempts_limit CHECK (attempts <= max_attempts),
    CONSTRAINT last_error_length CHECK (last_error IS NULL OR char_length(last_error) BETWEEN 1 AND 2000)
);

CREATE INDEX idx_outbox_events_claim ON outbox_events(next_attempt_at ASC, id ASC)
WHERE
    processed_at IS NULL AND attempts < max_attempts;

CREATE INDEX idx_outbox_events_cleanup ON outbox_events(processed_at ASC)
WHERE
    processed_at IS NOT NULL;

CREATE TABLE users(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    verified_at timestamp with time zone DEFAULT NULL,
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

CREATE TABLE user_profiles(
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    display_name CITEXT NOT NULL,
    avatar_url text,
    CONSTRAINT display_name_length CHECK (char_length(display_name) BETWEEN 3 AND 32),
    CONSTRAINT avatar_url_length CHECK (char_length(avatar_url) BETWEEN 3 AND 2048)
);

CREATE OR REPLACE VIEW user_aggregates AS
SELECT
    u.id,
    u.email,
    u.username,
    u.password_hash,
    u.preferred_presence,
    u.verified_at,
    p.display_name,
    p.avatar_url,
    u.created_at,
    u.updated_at,
    p.created_at AS profile_created_at,
    p.updated_at AS profile_updated_at
FROM
    users u
    INNER JOIN user_profiles p ON u.id = p.user_id;

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

CREATE TABLE channels(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    type SMALLINT NOT NULL,
    name text,
    CONSTRAINT type_values CHECK (type IN (0, 1, 2, 3, 4)),
    CONSTRAINT name_rules CHECK ((type = 1 AND name IS NULL) OR (type != 1 AND (name IS NULL OR length(trim(name)) BETWEEN 1 AND 100)))
);

CREATE TABLE channel_members(
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_user_id ON channel_members(user_id);

CREATE TABLE direct_message_channels(
    user1_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE UNIQUE,
    PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT dm_user_order CHECK (user1_id < user2_id)
);

CREATE TABLE relationships(
    user1_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    variant smallint NOT NULL,
    PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT actor_must_be_participant CHECK (actor_id IN (user1_id, user2_id)),
    CONSTRAINT relationship_values CHECK (variant IN (1, 2, 3))
);

CREATE INDEX idx_relationships_u1_perf ON relationships(user1_id, variant, actor_id) INCLUDE (created_at, updated_at);

CREATE INDEX idx_relationships_u2_perf ON relationships(user2_id, variant, actor_id) INCLUDE (created_at, updated_at);

CREATE OR REPLACE VIEW relationship_perspectives WITH ( security_invoker = TRUE
) AS
SELECT
    r.user1_id AS user_id,
    r.user2_id AS peer_id,
    r.variant,
    r.actor_id,
(
        r.actor_id = r.user1_id
) AS is_initiator,
    r.created_at,
    r.updated_at,
    u.username,
    up.display_name,
    up.avatar_url,
    u.preferred_presence AS user_preferred_presence,
    dm.channel_id
FROM
    relationships r
    JOIN users u ON u.id = r.user2_id
    LEFT JOIN user_profiles up ON up.user_id = r.user2_id
    LEFT JOIN direct_message_channels dm ON dm.user1_id = r.user1_id
        AND dm.user2_id = r.user2_id
WHERE
    r.variant != 3
    OR r.actor_id = r.user1_id
UNION ALL
SELECT
    r.user2_id AS user_id,
    r.user1_id AS peer_id,
    r.variant,
    r.actor_id,
(
        r.actor_id = r.user2_id
) AS is_initiator,
    r.created_at,
    r.updated_at,
    u.username,
    up.display_name,
    up.avatar_url,
    u.preferred_presence AS user_preferred_presence,
    dm.channel_id
FROM
    relationships r
    JOIN users u ON u.id = r.user1_id
    LEFT JOIN user_profiles up ON up.user_id = r.user1_id
    LEFT JOIN direct_message_channels dm ON dm.user1_id = r.user1_id
        AND dm.user2_id = r.user2_id
WHERE
    r.variant != 3
    OR r.actor_id = r.user2_id;

