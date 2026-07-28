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

-- Enums for Channel Types: 0: DM, 1: GROUP_DM
CREATE TABLE channels(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    owner_id uuid REFERENCES users(id) ON DELETE SET NULL,
    last_message_id uuid DEFAULT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type smallint NOT NULL,
    name text DEFAULT NULL,
    icon_url text DEFAULT NULL,
    CONSTRAINT channel_rules CHECK ((type = 0 AND owner_id IS NULL AND name IS NULL AND icon_url IS NULL) OR (type = 1 AND (name IS NULL OR length(trim(name)) BETWEEN 1 AND 100) AND (icon_url IS NULL OR length(icon_url) BETWEEN 3 AND 2048)))
);

CREATE INDEX idx_channels_owner_id ON channels(owner_id)
WHERE
    owner_id IS NOT NULL;

CREATE INDEX idx_channels_last_message_id ON channels(last_message_id DESC)
WHERE
    last_message_id IS NOT NULL;

CREATE TABLE messages(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id uuid REFERENCES users(id) ON DELETE SET NULL,
    reply_to_message_id uuid REFERENCES messages(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at timestamptz DEFAULT NULL,
    is_pinned boolean NOT NULL DEFAULT FALSE,
    content text,
    CONSTRAINT content_validity CHECK (content IS NULL OR length(trim(content)) BETWEEN 1 AND 4000)
);

CREATE INDEX idx_messages_channel_created_id ON messages(channel_id, created_at DESC, id DESC);

CREATE INDEX idx_messages_pinned ON messages(channel_id, created_at DESC)
WHERE
    is_pinned = TRUE;

CREATE INDEX idx_messages_reply_to ON messages(reply_to_message_id, created_at ASC, id ASC)
WHERE
    reply_to_message_id IS NOT NULL;

ALTER TABLE channels
    ADD CONSTRAINT fk_channels_last_message FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL;

CREATE TABLE channel_members(
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id uuid REFERENCES messages(id) ON DELETE SET NULL,
    joined_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    mention_count integer NOT NULL DEFAULT 0,
    PRIMARY KEY (channel_id, user_id),
    CONSTRAINT mention_count_positive CHECK (mention_count >= 0)
);

CREATE INDEX idx_channel_members_user ON channel_members(user_id);

CREATE INDEX idx_channel_members_last_read ON channel_members(last_read_message_id)
WHERE
    last_read_message_id IS NOT NULL;

CREATE TABLE message_attachments(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_name text NOT NULL,
    file_size integer NOT NULL, -- Size in bytes (supports files up to 2GB, or use bigint if larger limits needed)
    content_type text NOT NULL, -- e.g., 'image/png', 'application/pdf'
    url text NOT NULL, -- CDN URL or object storage path (e.g., S3 key or public URL)
    width integer DEFAULT NULL, -- Populated if the file is an image/video
    height integer DEFAULT NULL, -- Populated if the file is an image/video
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT file_name_validity CHECK (length(trim(file_name)) BETWEEN 1 AND 255),
    CONSTRAINT file_size_positive CHECK (file_size > 0),
    CONSTRAINT content_type_validity CHECK (length(trim(content_type)) BETWEEN 1 AND 128),
    CONSTRAINT url_validity CHECK (length(trim(url)) BETWEEN 3 AND 2048)
);

CREATE INDEX idx_message_attachments_message_id ON message_attachments(message_id);

CREATE TABLE message_reactions(
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    emoji text NOT NULL,
    PRIMARY KEY (message_id, user_id, emoji),
    CONSTRAINT emoji_length CHECK (char_length(trim(emoji)) BETWEEN 1 AND 64)
);

CREATE INDEX idx_reactions_message ON message_reactions(message_id);

CREATE TABLE dm_relationships(
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
    LEFT JOIN dm_relationships dm ON dm.user1_id = r.user1_id
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
    LEFT JOIN dm_relationships dm ON dm.user1_id = r.user1_id
        AND dm.user2_id = r.user2_id
WHERE
    r.variant != 3
    OR r.actor_id = r.user2_id;

