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

CREATE OR REPLACE FUNCTION check_channel_member_limit()
    RETURNS TRIGGER
    AS $$
DECLARE
    member_count integer;
BEGIN
    -- Lock parent channel row to serialize concurrent member additions
    PERFORM
        1
    FROM
        channels
    WHERE
        id = NEW.channel_id
    FOR UPDATE;
    SELECT
        COUNT(*)
    INTO
        member_count
    FROM
        channel_members
    WHERE
        channel_id = NEW.channel_id;
    IF member_count >= 10 THEN
        RAISE EXCEPTION 'channel member limit reached (max 10)'
            USING ERRCODE = 'check_violation';
    END IF;
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
    disabled_at timestamptz DEFAULT NULL,
    delete_scheduled_at timestamptz DEFAULT NULL,
    preferred_presence_until timestamptz DEFAULT NULL,
    preferred_presence smallint DEFAULT NULL,
    email CITEXT NOT NULL UNIQUE,
    username CITEXT NOT NULL UNIQUE,
    phone text DEFAULT NULL UNIQUE,
    password_hash text NOT NULL,
    CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 3 AND 32),
    CONSTRAINT username_reserved CHECK (lower(username) NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire')),
    CONSTRAINT password_hash_length CHECK (char_length(password_hash) BETWEEN 3 AND 255),
    CONSTRAINT preferred_presence_values CHECK (preferred_presence IN (4, 5, 6)),
    CONSTRAINT valid_standing CHECK (standing IN (0, 1)),
    CONSTRAINT valid_e164_phone CHECK (phone IS NULL OR phone ~ '^\+[1-9]\d{1,14}$')
);

CREATE TABLE user_profiles(
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    display_name citext NOT NULL,
    avatar_url text DEFAULT NULL,
    banner_color text DEFAULT NULL,
    CONSTRAINT display_name_length CHECK (char_length(display_name) BETWEEN 3 AND 32),
    CONSTRAINT avatar_url_length CHECK (char_length(avatar_url) BETWEEN 3 AND 2048),
    CONSTRAINT valid_hex_banner_color CHECK (banner_color IS NULL OR banner_color ~* '^#[0-9a-f]{6}$')
);

-- 1. Main MFA State & Secret Storage
CREATE TABLE user_mfa(
    -- 16-byte fixed alignment (UUIDs)
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- 8-byte fixed alignment (Timestamps)
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    enabled_at timestamptz DEFAULT NULL, -- NULL = setup initiated but not confirmed
    -- Variable length text / encrypted binary
    -- The base32 TOTP secret string (e.g., "JBSWY3DPEHPK3PXP")
    -- CRITICAL: Store this ENCRYPTED at the application layer using AES-GCM
    secret text NOT NULL
);

-- 2. Single-use Backup / Recovery Codes
CREATE TABLE user_mfa_backup_codes(
    -- 16-byte fixed alignment (UUIDs)
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- 8-byte fixed alignment (Timestamps)
    used_at timestamptz DEFAULT NULL, -- NULL = active/unused, timestamp = burned code
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Hash of the 8-to-10 character backup code (e.g. Argon2id or bcrypt hash)
    code_hash text NOT NULL,
    -- Constraints
    CONSTRAINT code_hash_length CHECK (length(code_hash) BETWEEN 10 AND 255)
);

-- Index for retrieving valid backup codes during emergency login
CREATE INDEX idx_user_mfa_backup_codes_user ON user_mfa_backup_codes(user_id)
WHERE
    used_at IS NULL;

CREATE OR REPLACE VIEW user_aggregates AS
SELECT
    u.id,
    u.email,
    u.username,
    u.phone,
    u.password_hash,
    u.preferred_presence,
    u.verified_at,
(m.enabled_at IS NOT NULL) AS mfa_enabled,
    p.display_name,
    p.avatar_url,
    p.banner_color,
    u.created_at,
    u.updated_at,
    p.created_at AS profile_created_at,
    p.updated_at AS profile_updated_at
FROM
    users u
    INNER JOIN user_profiles p ON u.id = p.user_id
    LEFT JOIN user_mfa m ON u.id = m.user_id;

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
    client text NOT NULL DEFAULT 'Unknown',
    CONSTRAINT refresh_token_hash_length CHECK (octet_length(refresh_token_hash) = 32),
    CONSTRAINT user_agent_length CHECK (length(user_agent) BETWEEN 1 AND 1000),
    CONSTRAINT os_length CHECK (length(os) BETWEEN 1 AND 100),
    CONSTRAINT client_length CHECK (length(client) BETWEEN 1 AND 100)
);

CREATE INDEX idx_sessions_user_active ON sessions(user_id)
WHERE (revoked_at IS NULL);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Enums for Channel Types: 0: DM, 1: GROUP_DM
CREATE TABLE channels(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    last_message_id uuid DEFAULT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_message_at timestamptz DEFAULT NULL,
    type smallint NOT NULL,
    name text DEFAULT NULL,
    icon_url text DEFAULT NULL,
    CONSTRAINT channel_rules CHECK ((type = 0 AND name IS NULL AND icon_url IS NULL) OR (type = 1 AND (name IS NULL OR length(trim(name)) BETWEEN 1 AND 100) AND (icon_url IS NULL OR length(icon_url) BETWEEN 3 AND 2048)))
);

-- Index for filtering/sorting active channels
CREATE INDEX idx_channels_last_message_id ON channels(last_message_id DESC)
WHERE
    last_message_id IS NOT NULL;

CREATE TABLE messages(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    reply_to_message_id uuid REFERENCES messages(id) ON DELETE SET NULL,
    author_id uuid REFERENCES users(id) ON DELETE SET NULL,
    forwarded_message_id uuid REFERENCES messages(id) ON DELETE SET NULL,
    forwarded_channel_id uuid REFERENCES channels(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at timestamptz DEFAULT NULL,
    pinned_at timestamptz DEFAULT NULL,
    -- 0: DEFAULT (Standard chat message)
    -- 1: RECIPIENT_ADD (User added to group)
    -- 2: RECIPIENT_REMOVE / MEMBER_LEAVE (User left or kicked)
    -- 3: CHANNEL_NAME_CHANGE (Group renamed)
    -- 4: CHANNEL_ICON_CHANGE (Group avatar updated)
    type smallint NOT NULL DEFAULT 0,
    content text,
    system_metadata jsonb DEFAULT NULL,
    CONSTRAINT content_validity CHECK (type != 0 OR (content IS NULL OR length(trim(content)) BETWEEN 1 AND 4000)),
    CONSTRAINT valid_forward CHECK ((forwarded_message_id IS NULL AND forwarded_channel_id IS NULL) OR (forwarded_message_id IS NOT NULL AND forwarded_channel_id IS NOT NULL))
);

-- 1. Main Cursor Pagination Index (Channel timeline loading: WHERE channel_id = $1 AND id < $2 ORDER BY id DESC)
CREATE INDEX idx_messages_channel_cursor ON messages(channel_id, id DESC);

-- 2. Partial Index for Pinned Messages (Avoids indexing unpinned messages)
CREATE INDEX idx_messages_pinned ON messages(channel_id, pinned_at DESC)
WHERE
    pinned_at IS NOT NULL;

-- 3. Reply Lookups (For fetching parent context when jumping to a reply)
CREATE INDEX idx_messages_reply_to ON messages(reply_to_message_id)
WHERE
    reply_to_message_id IS NOT NULL;

CREATE OR REPLACE VIEW message_user_aggregates AS
SELECT
    m.id,
    m.channel_id,
    m.reply_to_message_id,
    m.author_id,
    ua.username AS author_username,
    ua.display_name AS author_display_name,
    ua.avatar_url AS author_avatar_url,
    ua.banner_color AS author_banner_color,
    m.created_at,
    m.updated_at,
    m.edited_at,
    m.pinned_at,
    m.content
FROM
    messages m
    LEFT JOIN user_aggregates ua ON ua.id = m.author_id;

ALTER TABLE channels
    ADD CONSTRAINT fk_channels_last_message FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED;

-- DM Visibility States: 0: HIDDEN, 1: VISIBLE
CREATE TABLE channel_members(
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id uuid REFERENCES messages(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    pinned_at timestamptz DEFAULT NULL,
    muted_until timestamptz DEFAULT NULL,
    mention_count integer NOT NULL DEFAULT 0,
    is_visible boolean NOT NULL DEFAULT TRUE,
    PRIMARY KEY (channel_id, user_id),
    CONSTRAINT mention_count_positive CHECK (mention_count >= 0)
);

-- 1. Secondary Index: Reverse lookup for User Sidebar & Gateway Boot Sync
-- Primary Key covers (channel_id, user_id). You need this index for queries filtering by user_id:
-- e.g., "Get all channels for User X"
CREATE INDEX idx_channel_members_user_sidebar ON channel_members(user_id, channel_id) INCLUDE (last_read_message_id, mention_count, is_visible, muted_until);

-- 2. Channel Member Roster / Sidebar Order
-- For listing members in a channel ordered by when they joined:
CREATE INDEX idx_channel_members_channel_roster ON channel_members(channel_id, created_at ASC, user_id);

CREATE TRIGGER enforce_channel_member_limit
    BEFORE INSERT ON channel_members
    FOR EACH ROW
    EXECUTE FUNCTION check_channel_member_limit();

CREATE TABLE channel_invites(
    channel_id uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    inviter_id uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz DEFAULT NULL, -- NULL = Never expires
    max_uses integer NOT NULL DEFAULT 0, -- 0 = Infinite uses
    uses integer NOT NULL DEFAULT 0,
    code varchar(16) PRIMARY KEY,
    CONSTRAINT positive_uses CHECK (uses >= 0),
    CONSTRAINT valid_max_uses CHECK (max_uses >= 0)
);

-- Index for listing all active invites for a given channel's settings UI
CREATE INDEX idx_channel_invites_channel_id ON channel_invites(channel_id);

CREATE TABLE message_attachments(
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    file_size integer NOT NULL, -- Size in bytes (supports files up to 2GB, or use bigint if larger limits needed)
    width integer DEFAULT NULL, -- Populated if the file is an image/video
    height integer DEFAULT NULL, -- Populated if the file is an image/video
    file_name text NOT NULL,
    content_type text NOT NULL, -- e.g., 'image/png', 'application/pdf'
    url text NOT NULL, -- CDN URL or object storage path (e.g., S3 key or public URL)
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

CREATE TABLE relationships(
    user1_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id uuid UNIQUE REFERENCES channels(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    variant smallint NOT NULL,
    PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT actor_must_be_participant CHECK (actor_id IN (user1_id, user2_id)),
    CONSTRAINT relationship_values CHECK (variant IN (1, 2, 3)),
    CONSTRAINT channel_required_for_friends CHECK (variant != 2 OR channel_id IS NOT NULL)
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
    r.channel_id,
    r.created_at,
    r.updated_at,
    u.username,
    up.display_name,
    up.avatar_url
FROM
    relationships r
    JOIN users u ON u.id = r.user2_id
    LEFT JOIN user_profiles up ON up.user_id = r.user2_id
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
    r.channel_id,
    r.created_at,
    r.updated_at,
    u.username,
    up.display_name,
    up.avatar_url
FROM
    relationships r
    JOIN users u ON u.id = r.user1_id
    LEFT JOIN user_profiles up ON up.user_id = r.user1_id
WHERE
    r.variant != 3
    OR r.actor_id = r.user2_id;

