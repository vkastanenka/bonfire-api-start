CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE outbox_events(
    id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_attempt_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lease_expires_at timestamptz DEFAULT NULL,
    processed_at timestamptz DEFAULT NULL,
    locked_by uuid DEFAULT NULL,
    aggregate_id uuid DEFAULT NULL,
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 5,
    event_type text NOT NULL,
    aggregate_type text DEFAULT NULL,
    trace_id text DEFAULT NULL,
    payload jsonb NOT NULL,
    last_error text DEFAULT NULL,
    CONSTRAINT outbox_events_pkey PRIMARY KEY (id, created_at),
    CONSTRAINT event_type_length CHECK (char_length(event_type) BETWEEN 1 AND 100),
    CONSTRAINT aggregate_type_length CHECK (aggregate_type IS NULL OR char_length(aggregate_type) BETWEEN 1 AND 100),
    CONSTRAINT trace_id_length CHECK (trace_id IS NULL OR char_length(trace_id) BETWEEN 1 AND 256),
    CONSTRAINT payload_populated CHECK (payload != '{}'::jsonb AND payload != '[]'::jsonb),
    CONSTRAINT payload_size_limit CHECK (pg_column_size(payload) < 102400),
    CONSTRAINT attempts_limit CHECK (attempts <= max_attempts),
    CONSTRAINT last_error_length CHECK (last_error IS NULL OR char_length(last_error) BETWEEN 1 AND 2000)
)
PARTITION BY RANGE (created_at);

CREATE INDEX idx_outbox_events_claim ON outbox_events(next_attempt_at ASC, id ASC) INCLUDE (lease_expires_at)
WHERE
    processed_at IS NULL AND attempts < max_attempts;

CREATE INDEX idx_outbox_events_dead_letter ON outbox_events(created_at DESC)
WHERE
    processed_at IS NULL AND attempts >= max_attempts;

CREATE TABLE users(
    id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at timestamptz DEFAULT NULL,
    disabled_at timestamptz DEFAULT NULL,
    delete_scheduled_at timestamptz DEFAULT NULL,
    preferred_presence_until timestamptz DEFAULT NULL,
    preferred_presence smallint DEFAULT NULL,
    email citext NOT NULL,
    username citext NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_username_key UNIQUE (username),
    CONSTRAINT users_phone_key UNIQUE (phone),
    CONSTRAINT preferred_presence_valid CHECK (preferred_presence IN (4, 5, 6)),
    CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 3 AND 32),
    CONSTRAINT username_valid CHECK (username NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire')),
    CONSTRAINT password_hash_length CHECK (char_length(password_hash) BETWEEN 50 AND 255),
    CONSTRAINT phone_valid CHECK (phone IS NULL OR phone ~ '^\+[1-9]\d{1,14}$')
);

CREATE INDEX idx_users_delete_scheduled_at ON users(delete_scheduled_at ASC, id ASC)
WHERE
    delete_scheduled_at IS NOT NULL;

CREATE TABLE user_profiles(
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    display_name citext NOT NULL,
    bio text DEFAULT NULL,
    avatar_url text DEFAULT NULL,
    banner_color text DEFAULT NULL,
    CONSTRAINT user_profiles_pkey PRIMARY KEY (user_id),
    CONSTRAINT fk_user_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT display_name_length CHECK (char_length(display_name) BETWEEN 1 AND 32),
    CONSTRAINT bio_length CHECK (char_length(bio) <= 190),
    CONSTRAINT avatar_url_length CHECK (char_length(avatar_url) BETWEEN 1 AND 2048),
    CONSTRAINT valid_hex_banner_color CHECK (banner_color IS NULL OR banner_color ~* '^#[0-9a-f]{6}$')
);

CREATE TABLE user_mfa(
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    enabled_at timestamptz DEFAULT NULL,
    last_used_step bigint NOT NULL DEFAULT 0,
    secret text NOT NULL,
    CONSTRAINT user_mfa_pkey PRIMARY KEY (user_id),
    CONSTRAINT fk_user_mfa_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT last_used_step_non_negative CHECK (last_used_step >= 0),
    CONSTRAINT secret_length CHECK (char_length(secret) BETWEEN 16 AND 512)
);

CREATE INDEX idx_user_mfa_enabled ON user_mfa(user_id)
WHERE
    enabled_at IS NOT NULL;

CREATE TABLE user_mfa_backup_codes(
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used_at timestamptz DEFAULT NULL,
    code_hash text NOT NULL,
    CONSTRAINT user_mfa_backup_codes_pkey PRIMARY KEY (id),
    CONSTRAINT fk_user_mfa_backup_codes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT code_hash_length CHECK (char_length(code_hash) BETWEEN 10 AND 255)
);

CREATE INDEX idx_user_mfa_backup_codes_active ON user_mfa_backup_codes(user_id)
WHERE
    used_at IS NULL;

CREATE INDEX idx_user_mfa_backup_codes_user_hash ON user_mfa_backup_codes(user_id, code_hash)
WHERE
    used_at IS NULL;

CREATE TABLE sessions(
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz DEFAULT NULL,
    client_ip inet NOT NULL,
    refresh_token_hash bytea NOT NULL,
    os text NOT NULL DEFAULT 'Unknown',
    client text NOT NULL DEFAULT 'Unknown',
    user_agent text NOT NULL,
    CONSTRAINT sessions_pkey PRIMARY KEY (id),
    CONSTRAINT sessions_refresh_token_hash_key UNIQUE (refresh_token_hash),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT refresh_token_hash_length CHECK (octet_length(refresh_token_hash) = 32),
    CONSTRAINT user_agent_length CHECK (char_length(user_agent) BETWEEN 1 AND 1000),
    CONSTRAINT os_length CHECK (char_length(os) BETWEEN 1 AND 100),
    CONSTRAINT client_length CHECK (char_length(client) BETWEEN 1 AND 100)
);

CREATE INDEX idx_sessions_cleanup ON sessions(expires_at ASC);

CREATE INDEX idx_sessions_user_active ON sessions(user_id, last_seen_at DESC)
WHERE
    revoked_at IS NULL;

CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE TABLE channels(
    id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type smallint NOT NULL,
    name text DEFAULT NULL,
    icon_url text DEFAULT NULL,
    CONSTRAINT channels_pkey PRIMARY KEY (id),
    CONSTRAINT valid_channel_type CHECK (type IN (1, 2)),
    CONSTRAINT channel_rules CHECK ((type = 1 AND name IS NULL AND icon_url IS NULL) OR (type = 2 AND (name IS NULL OR char_length(trim(name)) BETWEEN 1 AND 100) AND (icon_url IS NULL OR char_length(icon_url) BETWEEN 3 AND 2048)))
);

CREATE TABLE messages(
    id uuid NOT NULL,
    channel_id uuid NOT NULL,
    author_id uuid DEFAULT NULL,
    reply_to_message_id uuid DEFAULT NULL,
    forwarded_message_id uuid DEFAULT NULL,
    forwarded_channel_id uuid DEFAULT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at timestamptz DEFAULT NULL,
    pinned_at timestamptz DEFAULT NULL,
    type smallint NOT NULL DEFAULT 1,
    content text DEFAULT NULL,
    system_metadata jsonb DEFAULT NULL,
    CONSTRAINT messages_pkey PRIMARY KEY (id),
    CONSTRAINT fk_messages_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_messages_reply_to FOREIGN KEY (reply_to_message_id) REFERENCES messages(id) ON DELETE SET NULL,
    CONSTRAINT fk_messages_forwarded_message FOREIGN KEY (forwarded_message_id) REFERENCES messages(id) ON DELETE SET NULL,
    CONSTRAINT fk_messages_forwarded_channel FOREIGN KEY (forwarded_channel_id) REFERENCES channels(id) ON DELETE SET NULL,
    CONSTRAINT valid_message_type CHECK (type IN (1, 2, 3, 4, 5)),
    CONSTRAINT content_length CHECK (content IS NULL OR char_length(trim(content)) BETWEEN 1 AND 4000),
    CONSTRAINT valid_forward CHECK ((forwarded_message_id IS NULL AND forwarded_channel_id IS NULL) OR (forwarded_message_id IS NOT NULL AND forwarded_channel_id IS NOT NULL))
);

CREATE INDEX idx_messages_channel_pagination ON messages(channel_id, id DESC);

CREATE INDEX idx_messages_pinned ON messages(channel_id, pinned_at DESC)
WHERE
    pinned_at IS NOT NULL;

CREATE INDEX idx_messages_reply_to ON messages(reply_to_message_id)
WHERE
    reply_to_message_id IS NOT NULL;

CREATE INDEX idx_messages_forwarded_msg ON messages(forwarded_message_id)
WHERE
    forwarded_message_id IS NOT NULL;

CREATE TABLE channel_members(
    channel_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_message_id uuid DEFAULT NULL,
    pinned_at timestamptz DEFAULT NULL,
    muted_until timestamptz DEFAULT NULL,
    mention_count integer NOT NULL DEFAULT 0,
    is_visible boolean NOT NULL DEFAULT TRUE,
    CONSTRAINT channel_members_pkey PRIMARY KEY (channel_id, user_id),
    CONSTRAINT fk_channel_members_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    CONSTRAINT fk_channel_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_channel_members_last_read_msg FOREIGN KEY (last_read_message_id) REFERENCES messages(id) ON DELETE SET NULL,
    CONSTRAINT mention_count_positive CHECK (mention_count >= 0)
);

CREATE INDEX idx_channel_members_user_sidebar ON channel_members(user_id, channel_id)
WHERE
    is_visible = TRUE;

CREATE INDEX idx_channel_members_channel_roster ON channel_members(channel_id, created_at ASC);

CREATE INDEX idx_channel_members_last_read_msg ON channel_members(last_read_message_id)
WHERE
    last_read_message_id IS NOT NULL;

CREATE TABLE channel_invites(
    code varchar(16) NOT NULL,
    channel_id uuid NOT NULL,
    inviter_id uuid DEFAULT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz DEFAULT NULL,
    max_uses integer NOT NULL DEFAULT 0,
    uses integer NOT NULL DEFAULT 0,
    CONSTRAINT channel_invites_pkey PRIMARY KEY (code),
    CONSTRAINT fk_channel_invites_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    CONSTRAINT fk_channel_invites_inviter FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT positive_uses CHECK (uses >= 0),
    CONSTRAINT valid_max_uses CHECK (max_uses >= 0),
    CONSTRAINT uses_within_max CHECK (max_uses = 0 OR uses <= max_uses)
);

CREATE INDEX idx_channel_invites_channel_id ON channel_invites(channel_id);

CREATE TABLE message_attachments(
    id uuid NOT NULL,
    message_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    file_size bigint NOT NULL,
    width integer DEFAULT NULL,
    height integer DEFAULT NULL,
    file_name text NOT NULL,
    content_type text NOT NULL,
    url text NOT NULL,
    CONSTRAINT message_attachments_pkey PRIMARY KEY (id),
    CONSTRAINT fk_message_attachments_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    CONSTRAINT file_name_validity CHECK (length(trim(file_name)) BETWEEN 1 AND 255),
    CONSTRAINT file_size_positive CHECK (file_size > 0),
    CONSTRAINT content_type_validity CHECK (length(trim(content_type)) BETWEEN 1 AND 128),
    CONSTRAINT url_validity CHECK (length(trim(url)) BETWEEN 3 AND 2048)
);

CREATE INDEX idx_message_attachments_message_id ON message_attachments(message_id, created_at ASC);

CREATE TABLE message_reactions(
    message_id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    emoji text NOT NULL,
    CONSTRAINT message_reactions_pkey PRIMARY KEY (message_id, user_id, emoji),
    CONSTRAINT fk_message_reactions_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_message_reactions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT emoji_length CHECK (char_length(trim(emoji)) BETWEEN 1 AND 64)
);

CREATE INDEX idx_message_reactions_message_id ON message_reactions(message_id, created_at ASC);

CREATE TABLE relationships(
    user1_id uuid NOT NULL,
    user2_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    channel_id uuid DEFAULT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    variant smallint NOT NULL,
    CONSTRAINT relationships_pkey PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT fk_relationships_user1 FOREIGN KEY (user1_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_relationships_user2 FOREIGN KEY (user2_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_relationships_actor FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_relationships_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    CONSTRAINT relationships_channel_key UNIQUE (channel_id),
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT actor_must_be_participant CHECK (actor_id IN (user1_id, user2_id)),
    CONSTRAINT relationship_values CHECK (variant IN (1, 2, 3)),
    CONSTRAINT channel_required_for_friends CHECK (variant != 2 OR channel_id IS NOT NULL)
);

CREATE INDEX idx_relationships_user1 ON relationships(user1_id, variant);

CREATE INDEX idx_relationships_user2 ON relationships(user2_id, variant);

CREATE OR REPLACE VIEW user_aggregates AS
SELECT
    u.id,
    u.username,
    u.email,
    u.phone,
    p.display_name,
    p.bio,
    p.avatar_url,
    p.banner_color,
    u.preferred_presence,
    u.preferred_presence_until,
    u.verified_at,
    u.disabled_at,
    u.delete_scheduled_at,
    u.created_at,
    u.updated_at
FROM
    users u
    INNER JOIN user_profiles p ON u.id = p.user_id;

