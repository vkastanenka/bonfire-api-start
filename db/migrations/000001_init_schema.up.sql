CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE outbox_events(
    id uuid PRIMARY KEY,
    aggregate_id uuid,
    locked_by uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    next_attempt_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lease_expires_at timestamptz,
    processed_at timestamptz,
    attempts integer NOT NULL DEFAULT 0 CONSTRAINT attempts_non_negative CHECK (attempts >= 0),
    max_attempts integer NOT NULL DEFAULT 5 CONSTRAINT max_attempts_positive CHECK (max_attempts > 0),
    event_type text NOT NULL CONSTRAINT event_type_length CHECK (char_length(trim(event_type)) BETWEEN 1 AND 100),
    aggregate_type text CONSTRAINT aggregate_type_length CHECK (aggregate_type IS NULL OR char_length(trim(aggregate_type)) BETWEEN 1 AND 100),
    trace_id text CONSTRAINT trace_id_length CHECK (trace_id IS NULL OR char_length(trim(trace_id)) BETWEEN 1 AND 256),
    payload jsonb NOT NULL CONSTRAINT payload_populated CHECK (payload != '{}'::jsonb AND payload != '[]'::jsonb),
    last_error text CONSTRAINT last_error_length CHECK (char_length(trim(last_error)) BETWEEN 1 AND 2000),
    CONSTRAINT payload_size_limit CHECK (octet_length(payload::text) < 102400),
    CONSTRAINT attempts_limit CHECK (attempts <= max_attempts)
);

CREATE INDEX idx_outbox_events_claim ON outbox_events(next_attempt_at, lease_expires_at, id) INCLUDE (locked_by)
WHERE
    processed_at IS NULL AND attempts < max_attempts;

CREATE INDEX idx_outbox_events_cleanup ON outbox_events(processed_at, id)
WHERE
    processed_at IS NOT NULL;

CREATE TABLE users(
    id uuid PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at timestamptz,
    disabled_at timestamptz,
    delete_scheduled_at timestamptz,
    preferred_presence_until timestamptz,
    preferred_presence smallint CONSTRAINT preferred_presence_valid CHECK (preferred_presence IN (4, 5, 6)),
    email citext NOT NULL CONSTRAINT users_email_key UNIQUE CONSTRAINT email_length CHECK (char_length(email) BETWEEN 3 AND 255),
    username citext NOT NULL CONSTRAINT users_username_key UNIQUE CONSTRAINT username_length CHECK (char_length(username) BETWEEN 3 AND 32) CONSTRAINT username_valid CHECK (username NOT IN ('admin', 'root', 'support', 'system', 'moderator', 'bonfire')),
    display_name citext NOT NULL CONSTRAINT display_name_length CHECK (char_length(display_name) BETWEEN 1 AND 32),
    password_hash text NOT NULL CONSTRAINT password_hash_length CHECK (char_length(password_hash) BETWEEN 50 AND 255),
    phone text CONSTRAINT phone_valid CHECK (phone ~ '^\+[1-9]\d{1,14}$'),
    bio text CONSTRAINT bio_length CHECK (char_length(bio) <= 190),
    avatar_url text CONSTRAINT avatar_url_length CHECK (char_length(avatar_url) <= 2048),
    banner_color text CONSTRAINT banner_color_valid CHECK (banner_color ~* '^#[0-9a-f]{6}$')
);

CREATE INDEX idx_users_delete_scheduled_at ON users(delete_scheduled_at)
WHERE
    delete_scheduled_at IS NOT NULL;

CREATE UNIQUE INDEX idx_users_phone_unique ON users(phone)
WHERE
    phone IS NOT NULL;

CREATE TABLE sessions(
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL CONSTRAINT fk_sessions_user REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    client_ip inet NOT NULL,
    refresh_token_hash bytea NOT NULL UNIQUE CONSTRAINT refresh_token_hash_length CHECK (octet_length(refresh_token_hash) = 32),
    os text NOT NULL DEFAULT 'Unknown' CONSTRAINT os_length CHECK (char_length(os) BETWEEN 1 AND 100),
    client text NOT NULL DEFAULT 'Unknown' CONSTRAINT client_length CHECK (char_length(client) BETWEEN 1 AND 100),
    user_agent text NOT NULL CONSTRAINT user_agent_length CHECK (char_length(user_agent) BETWEEN 1 AND 1000)
);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE INDEX idx_sessions_user_active ON sessions(user_id, last_seen_at DESC)
WHERE
    revoked_at IS NULL;

CREATE TABLE channels(
    id uuid PRIMARY KEY,
    last_message_id uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_message_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type smallint NOT NULL DEFAULT 1 CONSTRAINT type_valid CHECK (type IN (1, 2)),
    name text CHECK (char_length(name) BETWEEN 1 AND 100),
    icon_url text CHECK (char_length(icon_url) BETWEEN 1 AND 2048)
);

CREATE INDEX idx_channels_last_message_id ON channels(last_message_id)
WHERE
    last_message_id IS NOT NULL;

CREATE TABLE channel_members(
    channel_id uuid NOT NULL CONSTRAINT fk_channel_members_channel REFERENCES channels(id) ON DELETE CASCADE,
    user_id uuid NOT NULL CONSTRAINT fk_channel_members_user REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id uuid,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_read_message_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    muted_until timestamptz,
    pinned_at timestamptz,
    mention_count int NOT NULL DEFAULT 0,
    is_visible boolean NOT NULL DEFAULT TRUE,
    CONSTRAINT channel_members_pkey PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_last_read_message_id ON channel_members(last_read_message_id)
WHERE
    last_read_message_id IS NOT NULL;

CREATE INDEX idx_channel_members_user_sidebar ON channel_members(user_id, pinned_at DESC NULLS LAST, channel_id)
WHERE
    is_visible = TRUE;

CREATE TABLE messages(
    id uuid PRIMARY KEY,
    channel_id uuid NOT NULL CONSTRAINT fk_messages_channel REFERENCES channels(id) ON DELETE CASCADE,
    author_id uuid CONSTRAINT fk_messages_author REFERENCES users(id) ON DELETE SET NULL,
    reply_to_message_id uuid CONSTRAINT fk_messages_reply_to REFERENCES messages(id) ON DELETE SET NULL,
    forward_message_id uuid CONSTRAINT fk_messages_forward_message REFERENCES messages(id) ON DELETE SET NULL,
    forward_channel_id uuid CONSTRAINT fk_messages_forward_channel REFERENCES channels(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at timestamptz,
    pinned_at timestamptz,
    type smallint NOT NULL DEFAULT 1 CONSTRAINT type_valid CHECK (type IN (1, 2, 3, 4, 5, 6, 7, 8, 9)),
    content text CONSTRAINT content_length CHECK (content IS NULL OR char_length(trim(content)) BETWEEN 1 AND 4000),
    metadata jsonb,
    CONSTRAINT forward_valid CHECK ((forward_message_id IS NULL AND forward_channel_id IS NULL) OR (forward_message_id IS NOT NULL AND forward_channel_id IS NOT NULL))
);

CREATE INDEX idx_messages_channel_id_id_desc ON messages(channel_id, id DESC);

CREATE INDEX idx_messages_author_id ON messages(author_id)
WHERE
    author_id IS NOT NULL;

CREATE INDEX idx_messages_reply_to_message_id ON messages(reply_to_message_id)
WHERE
    reply_to_message_id IS NOT NULL;

CREATE INDEX idx_messages_forward_message_id ON messages(forward_message_id)
WHERE
    forward_message_id IS NOT NULL;

CREATE INDEX idx_messages_forward_channel_id ON messages(forward_channel_id)
WHERE
    forward_channel_id IS NOT NULL;

CREATE INDEX idx_messages_pinned_at_desc_id_desc ON messages(channel_id, pinned_at DESC, id DESC)
WHERE
    pinned_at IS NOT NULL;

ALTER TABLE channels
    ADD CONSTRAINT fk_channels_last_message_id FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL;

ALTER TABLE channel_memberss
    ADD CONSTRAINT fk_channels_last_read_message_id FOREIGN KEY (last_read_message_id) REFERENCES messages(id) ON DELETE SET NULL;

CREATE TABLE message_reactions(
    message_id uuid NOT NULL CONSTRAINT fk_message_reactions_message REFERENCES messages(id) ON DELETE CASCADE,
    user_id uuid NOT NULL CONSTRAINT fk_message_reactions_user REFERENCES users(id) ON DELETE CASCADE,
    emoji text NOT NULL CONSTRAINT emoji_length CHECK (char_length(trim(emoji)) BETWEEN 1 AND 64),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT message_reactions_pkey PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_message_reactions_message_id_created_at ON message_reactions(message_id, created_at);

CREATE INDEX idx_message_reactions_user_id ON message_reactions(user_id);

CREATE TABLE message_attachments(
    id uuid PRIMARY KEY,
    message_id uuid NOT NULL CONSTRAINT fk_message_attachments_message REFERENCES messages(id) ON DELETE CASCADE,
    file_name text NOT NULL CONSTRAINT file_name_validity CHECK (length(trim(file_name)) BETWEEN 1 AND 255),
    content_type text NOT NULL CONSTRAINT content_type_validity CHECK (length(trim(content_type)) BETWEEN 1 AND 128),
    url text NOT NULL CONSTRAINT url_validity CHECK (length(trim(url)) BETWEEN 3 AND 2048),
    file_size bigint NOT NULL CONSTRAINT file_size_positive CHECK (file_size > 0),
    width integer,
    height integer,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_message_attachments_message_id ON message_attachments(message_id, id);

CREATE TABLE relations(
    user1_id uuid NOT NULL CONSTRAINT fk_relations_user1 REFERENCES users(id) ON DELETE CASCADE,
    user2_id uuid NOT NULL CONSTRAINT fk_relations_user2 REFERENCES users(id) ON DELETE CASCADE,
    actor_id uuid NOT NULL CONSTRAINT fk_relations_actor REFERENCES users(id) ON DELETE CASCADE,
    channel_id uuid CONSTRAINT fk_relations_channel REFERENCES channels(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    type smallint NOT NULL CONSTRAINT type_valid CHECK (type IN (1, 2, 3)),
    CONSTRAINT relations_pkey PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT user_order CHECK (user1_id < user2_id),
    CONSTRAINT actor_must_be_participant CHECK (actor_id IN (user1_id, user2_id)),
    CONSTRAINT channel_required_for_friends CHECK (type != 2 OR channel_id IS NOT NULL)
);

CREATE INDEX idx_relations_user1_type_created ON relations(user1_id, type, created_at DESC);

CREATE INDEX idx_relations_user2_type_created ON relations(user2_id, type, created_at DESC);

CREATE UNIQUE INDEX idx_relations_channel_id ON relations(channel_id)
WHERE
    channel_id IS NOT NULL;

CREATE INDEX idx_relations_user1_blocks ON relations(user1_id, user2_id)
WHERE
    type = 3;

CREATE INDEX idx_relations_user2_blocks ON relations(user2_id, user1_id)
WHERE
    type = 3;

