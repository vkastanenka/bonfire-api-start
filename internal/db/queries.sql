-- name: UserCreateAggregate :exec
WITH new_user AS (
INSERT INTO users(id, created_at, updated_at, verified_at, preferred_presence, email, username, password_hash)
        VALUES (@user_id, @user_created_at, @user_updated_at, @verified_at, @preferred_presence, @email, @username, @password_hash))
    INSERT INTO user_profiles(user_id, created_at, updated_at, display_name, avatar_url)
        VALUES (@user_id, @profile_created_at, @profile_updated_at, @display_name, @avatar_url);

-- name: UserGet :one
SELECT
    *
FROM
    user_aggregates
WHERE
    id = $1
LIMIT 1;

-- name: UserGetByEmail :one
SELECT
    *
FROM
    user_aggregates
WHERE
    email = $1
LIMIT 1;

-- name: UserGetByUsername :one
SELECT
    *
FROM
    user_aggregates
WHERE
    username = $1
LIMIT 1;

-- name: UserCheckAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            users.email = $1)::boolean AS email_available,
    NOT EXISTS (
        SELECT
            1
        FROM
            users
        WHERE
            users.username = $2)::boolean AS username_available;

-- name: UserUpdate :one
UPDATE
    users
SET
    email = @email,
    username = @username,
    password_hash = @password_hash,
    preferred_presence = @preferred_presence,
    verified_at = @verified_at,
    updated_at = @updated_at
WHERE
    id = @id
RETURNING
    *;

-- name: UserProfileUpsert :one
INSERT INTO user_profiles(user_id, created_at, updated_at, display_name, avatar_url)
    VALUES (@user_id, @created_at, @updated_at, @display_name, @avatar_url)
ON CONFLICT (user_id)
    DO UPDATE SET
        display_name = EXCLUDED.display_name,
        avatar_url = EXCLUDED.avatar_url,
        updated_at = EXCLUDED.updated_at
    RETURNING
        *;

-- name: SessionCreate :one
INSERT INTO sessions(id, user_id, refresh_token_hash, expires_at, revoked_at, client_ip, user_agent, os, browser, last_seen_at, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING
    *;

-- name: SessionGet :one
SELECT
    *
FROM
    sessions
WHERE
    id = $1
LIMIT 1;

-- name: SessionSave :one
UPDATE
    sessions
SET
    refresh_token_hash = $2,
    expires_at = $3,
    last_seen_at = $4,
    revoked_at = $5,
    updated_at = $6
WHERE
    id = $1
RETURNING
    *;

-- name: SessionDelete :exec
DELETE FROM sessions
WHERE id = $1;

-- name: SessionDeleteAllByUserID :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: SessionDeleteAllExcept :exec
DELETE FROM sessions
WHERE user_id = $1
    AND id != $2;

-- name: SessionDeleteAllExpired :exec
DELETE FROM sessions
WHERE expires_at <= CURRENT_TIMESTAMP;

-- name: OutboxEventCreate :one
INSERT INTO outbox_events(id, locked_by, created_at, updated_at, next_attempt_at, lease_expires_at, processed_at, attempts, max_attempts, event_type, last_error, payload)
    VALUES (@id, @locked_by, @created_at, @updated_at, @next_attempt_at, @lease_expires_at, @processed_at, @attempts, @max_attempts, @event_type, @last_error, @payload)
RETURNING
    *;

-- name: OutboxEventGet :one
SELECT
    *
FROM
    outbox_events
WHERE
    id = $1;

-- name: OutboxEventList :many
SELECT
    *
FROM
    outbox_events
WHERE (sqlc.narg('cursor_id')::uuid IS NULL
    OR id < sqlc.narg('cursor_id'))
ORDER BY
    id DESC
LIMIT @result_limit;

-- name: OutboxEventAcquireBatch :many
UPDATE
    outbox_events
SET
    locked_by = sqlc.arg(worker_id)::uuid,
    lease_expires_at = CURRENT_TIMESTAMP + make_interval(secs => sqlc.arg(lease_duration_seconds)::int),
    updated_at = CURRENT_TIMESTAMP
WHERE
    id IN (
        SELECT
            id
        FROM
            outbox_events
        WHERE
            processed_at IS NULL
            AND attempts < max_attempts
            AND next_attempt_at <= CURRENT_TIMESTAMP
            AND (lease_expires_at IS NULL
                OR lease_expires_at < CURRENT_TIMESTAMP)
        ORDER BY
            next_attempt_at ASC,
            id ASC
        LIMIT sqlc.arg(batch_size)::int
        FOR UPDATE
            SKIP LOCKED)
RETURNING
    *;

-- name: OutboxEventUpdate :one
UPDATE
    outbox_events
SET
    locked_by = sqlc.arg(locked_by),
    lease_expires_at = sqlc.arg(lease_expires_at),
    processed_at = sqlc.arg(processed_at),
    attempts = sqlc.arg(attempts),
    max_attempts = sqlc.arg(max_attempts),
    next_attempt_at = sqlc.arg(next_attempt_at),
    last_error = sqlc.arg(last_error),
    updated_at = sqlc.arg(updated_at)
WHERE
    id = sqlc.arg(id)
RETURNING
    *;

-- name: OutboxEventRenewLease :exec
UPDATE
    outbox_events
SET
    lease_expires_at = CURRENT_TIMESTAMP + make_interval(secs => sqlc.arg(lease_duration_seconds)::int),
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = sqlc.arg(id)::uuid
    AND locked_by = sqlc.arg(worker_id)::uuid
    AND processed_at IS NULL;

-- name: OutboxEventDelete :exec
DELETE FROM outbox_events
WHERE id = $1;

-- name: OutboxEventPurgeProcessed :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - make_interval(days => sqlc.arg(retention_days)::int));

-- ============================================================================
-- CHANNELS
-- ============================================================================
-- name: ChannelCreate :one
INSERT INTO channels(id, created_at, updated_at, type, last_message_id, name, icon_url)
    VALUES (@id::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint, sqlc.narg('last_message_id')::uuid, sqlc.narg('name')::text, sqlc.narg('icon_url')::text)
RETURNING
    *;

-- name: ChannelDelete :exec
DELETE FROM channels
WHERE id = @id::uuid;

-- name: ChannelGet :one
SELECT
    *
FROM
    channels
WHERE
    id = @id::uuid;

-- name: ChannelGetForMember :one
SELECT
    c.*
FROM
    channels c
    INNER JOIN channel_members cm ON cm.channel_id = c.id
WHERE
    c.id = @channel_id::uuid
    AND cm.user_id = @user_id::uuid;

-- name: ChannelHasMessagesAfter :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            messages
        WHERE
            channel_id = @channel_id::uuid
            AND (created_at > @created_at::timestamptz
                OR (created_at = @created_at::timestamptz
                    AND id > @message_id::uuid)));

-- name: ChannelHasMessagesBefore :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            messages
        WHERE
            channel_id = @channel_id::uuid
            AND (created_at < @created_at::timestamptz
                OR (created_at = @created_at::timestamptz
                    AND id < @message_id::uuid)));

-- name: ChannelListByUser :many
-- Used for populating the user's sidebar with channel info & peer profiles for DMs
SELECT
    -- 1. Primary Identifiers & References
    cm.channel_id,
    cm.user_id,
    rp.peer_id AS peer_user_id,
    c.type AS channel_type,
    -- 2. Display & Metadata (Resolved for 1:1 DMs or Group DMs)
    COALESCE(c.name, rp.display_name, rp.username) AS channel_name,
    COALESCE(c.icon_url, rp.avatar_url) AS channel_icon_url,
    -- 3. Read State, Activity & Metrics
    c.last_message_id AS channel_last_message_id,
    cm.last_read_message_id,
    cm.mention_count,
    -- 4. Timestamps
    cm.created_at,
    cm.last_read_at,
    c.updated_at AS channel_updated_at
FROM
    channel_members cm
    JOIN channels c ON cm.channel_id = c.id
    -- Join relationship_perspectives ONLY for 1:1 DMs to resolve peer profile
    LEFT JOIN relationship_perspectives rp ON c.type = 0
        AND rp.user_id = cm.user_id
        AND rp.channel_id = c.id
WHERE
    cm.user_id = @user_id::uuid
ORDER BY
    c.updated_at DESC,
    c.id ASC;

-- name: ChannelUpdate :one
UPDATE
    channels
SET
    name = COALESCE(sqlc.narg('name')::text, name),
    icon_url = COALESCE(sqlc.narg('icon_url')::text, icon_url),
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    *;

-- name: ChannelUpdateLastMessage :one
UPDATE
    channels
SET
    last_message_id = @last_message_id::uuid,
    updated_at = @updated_at::timestamptz
WHERE
    id = @channel_id::uuid
    AND updated_at <= @updated_at::timestamptz
RETURNING
    *;

-- ============================================================================
-- CHANNEL MEMBERS
-- ============================================================================
-- name: ChannelMemberAddBatch :exec
INSERT INTO channel_members(channel_id, user_id, created_at, updated_at, last_read_at, mention_count, last_read_message_id)
SELECT
    @channel_id::uuid,
    u.user_id,
    u.created_at,
    u.updated_at,
    u.last_read_at,
    u.mention_count,
    u.last_read_message_id
FROM
    ROWS
FROM (unnest(@user_ids::uuid[]),
    unnest(@created_ats::timestamptz[]),
    unnest(@updated_ats::timestamptz[]),
    unnest(@last_read_ats::timestamptz[]),
    unnest(@mention_counts::int[]),
    unnest(@last_read_message_ids::uuid[])) AS u(user_id,
        created_at,
        updated_at,
        last_read_at,
        mention_count,
        last_read_message_id)
ON CONFLICT (channel_id,
    user_id)
    DO UPDATE SET
        updated_at = EXCLUDED.updated_at,
        last_read_at = EXCLUDED.last_read_at,
        mention_count = EXCLUDED.mention_count,
        last_read_message_id = COALESCE(EXCLUDED.last_read_message_id, channel_members.last_read_message_id);

-- name: ChannelMemberGet :one
SELECT
    *
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberGetUnreadCount :one
SELECT
    COUNT(*)::int AS unread_count
FROM
    messages m
    JOIN channel_members cm ON cm.channel_id = m.channel_id
        AND cm.user_id = @user_id::uuid
WHERE
    m.channel_id = @channel_id::uuid
    AND m.created_at > cm.last_read_at;

-- name: ChannelMemberIncrementMentionCountBatch :exec
UPDATE
    channel_members
SET
    mention_count = mention_count + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = ANY (@user_ids::uuid[]);

-- name: ChannelMemberListByChannel :many
SELECT
    cm.channel_id,
    cm.user_id,
    cm.created_at,
    cm.updated_at,
    cm.last_read_at,
    cm.mention_count,
    cm.last_read_message_id,
    ua.username,
    ua.display_name,
    ua.avatar_url
FROM
    channel_members cm
    JOIN user_aggregates ua ON ua.id = cm.user_id
WHERE
    cm.channel_id = @channel_id::uuid
ORDER BY
    cm.created_at ASC,
    cm.user_id ASC;

-- name: ChannelMemberRemove :exec
DELETE FROM channel_members
WHERE channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberResetMentionCount :exec
UPDATE
    channel_members
SET
    mention_count = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberUpdateLastRead :exec
UPDATE
    channel_members
SET
    last_read_message_id = CASE WHEN @last_read_at::timestamptz > last_read_at THEN
        COALESCE(sqlc.narg('last_read_message_id')::uuid, last_read_message_id)
    ELSE
        last_read_message_id
    END,
    last_read_at = GREATEST(last_read_at, @last_read_at::timestamptz),
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- ============================================================================
-- MESSAGES: TODO
-- ============================================================================
-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, reply_to_message_id, author_id, created_at, edited_at, is_pinned, content)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('author_id')::uuid, @created_at::timestamptz, sqlc.narg('edited_at')::timestamptz, @is_pinned::boolean, sqlc.narg('content')::text)
RETURNING
    *;

-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = @id::uuid;

-- name: MessageGet :one
SELECT
    *
FROM
    messages
WHERE
    id = @id::uuid;

-- name: MessageGetFirstUnread :one
-- Fetches the first message created after the user's last_read_at timestamp
SELECT
    m.*
FROM
    messages m
    JOIN channel_members cm ON cm.channel_id = m.channel_id
        AND cm.user_id = @user_id::uuid
WHERE
    m.channel_id = @channel_id::uuid
    AND m.created_at > cm.last_read_at
ORDER BY
    m.created_at ASC,
    m.id ASC
LIMIT 1;

-- name: MessageGetLatest :one
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
ORDER BY
    created_at DESC,
    id DESC
LIMIT 1;

-- name: MessageListByChannelAfter :many
-- Fetches newer messages using keyset pagination.
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
    AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
        OR (created_at,
            id) >(sqlc.narg('cursor_created_at')::timestamptz,
            sqlc.narg('cursor_id')::uuid))
ORDER BY
    created_at ASC,
    id ASC
LIMIT @result_limit::int;

-- name: MessageListByChannelAround :many
WITH around_window AS ((
        SELECT
            m1.*
        FROM
            messages m1
        WHERE
            m1.channel_id = @channel_id::uuid
            AND (m1.created_at,
                m1.id) <=(@cursor_created_at::timestamptz,
                @cursor_id::uuid)
        ORDER BY
            m1.created_at DESC,
            m1.id DESC
        LIMIT @older_limit::int)
UNION ALL (
    SELECT
        m2.*
    FROM
        messages m2
    WHERE
        m2.channel_id = @channel_id::uuid
        AND (m2.created_at,
            m2.id) >(@cursor_created_at::timestamptz,
            @cursor_id::uuid)
    ORDER BY
        m2.created_at ASC,
        m2.id ASC
    LIMIT @newer_limit::int))
SELECT
    m.*
FROM
    around_window aw
    JOIN messages m ON m.id = aw.id
ORDER BY
    aw.created_at ASC,
    aw.id ASC;

-- name: MessageListByChannelBefore :many
-- Fetches older messages using keyset pagination.
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
    AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
        OR (created_at,
            id) <(sqlc.narg('cursor_created_at')::timestamptz,
            sqlc.narg('cursor_id')::uuid))
ORDER BY
    created_at DESC,
    id DESC
LIMIT @result_limit::int;

-- name: MessageListPinnedByChannel :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id::uuid
    AND is_pinned = TRUE
ORDER BY
    created_at DESC,
    id DESC;

-- name: MessageListReplies :many
SELECT
    m.*
FROM
    messages m
WHERE
    m.reply_to_message_id = @reply_to_message_id::uuid
ORDER BY
    m.created_at ASC,
    m.id ASC;

-- name: MessageSetPinned :one
UPDATE
    messages
SET
    is_pinned = @is_pinned::boolean
WHERE
    id = @id::uuid
RETURNING
    *;

-- name: MessageUpdateContent :one
UPDATE
    messages
SET
    content = sqlc.narg('content')::text,
    edited_at = @edited_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    *;

-- ============================================================================
-- MESSAGE ATTACHMENTS: TODO
-- ============================================================================
-- name: AttachmentCreateBatch :exec
INSERT INTO message_attachments(id, message_id, file_name, file_size, content_type, url, width, height, created_at)
SELECT
    unnest(@ids::uuid[]),
    @message_id,
    unnest(@file_names::text[]),
    unnest(@file_sizes::int[]),
    unnest(@content_types::text[]),
    unnest(@urls::text[]),
    unnest(@widths::int[]),
    unnest(@heights::int[]),
    unnest(@created_ats::timestamptz[])
WHERE
    @message_id IS NOT NULL;

-- name: AttachmentListByMessage :many
SELECT
    *
FROM
    message_attachments
WHERE
    message_id = $1
ORDER BY
    created_at ASC;

-- name: AttachmentListByMessagesBatch :many
-- Highly efficient for bulk-loading attachments when fetching a page of messages
SELECT
    *
FROM
    message_attachments
WHERE
    message_id = ANY (@message_ids::uuid[])
ORDER BY
    created_at ASC;

-- name: AttachmentDeleteByMessage :exec
DELETE FROM message_attachments
WHERE message_id = $1;

-- ============================================================================
-- MESSAGE REACTIONS: TODO
-- ============================================================================
-- name: ReactionAdd :one
INSERT INTO message_reactions(message_id, user_id, emoji)
    VALUES (@message_id, @user_id, @emoji)
ON CONFLICT (message_id, user_id, emoji)
    DO UPDATE SET
        emoji = EXCLUDED.emoji
    RETURNING
        *;

-- name: ReactionRemove :exec
DELETE FROM message_reactions
WHERE message_id = @message_id
    AND user_id = @user_id
    AND emoji = @emoji;

-- name: ReactionListByMessage :many
SELECT
    *
FROM
    message_reactions
WHERE
    message_id = $1
ORDER BY
    created_at ASC;

-- name: ReactionSummarizeByMessage :many
SELECT
    emoji,
    COUNT(*)::int AS count,
    BOOL_OR(user_id = @current_user_id) AS me_reacted
FROM
    message_reactions
WHERE
    message_id = @message_id
GROUP BY
    emoji
ORDER BY
    count DESC;

-- ============================================================================
-- RELATIONSHIPS
-- ============================================================================
-- name: RelationshipGet :one
SELECT
    *
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationshipGetByChannelID :one
SELECT
    *
FROM
    relationships
WHERE
    channel_id = @channel_id::uuid;

-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, actor_id, variant, created_at, updated_at, channel_id)
    VALUES (LEAST(@user1_id::uuid, @user2_id::uuid), GREATEST(@user1_id::uuid, @user2_id::uuid), @actor_id, @variant, @created_at, @updated_at, sqlc.narg('channel_id')::uuid)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        variant = EXCLUDED.variant,
        actor_id = EXCLUDED.actor_id,
        updated_at = EXCLUDED.updated_at,
        channel_id = COALESCE(EXCLUDED.channel_id, relationships.channel_id)
    RETURNING
        *;

-- name: RelationshipDelete :exec
DELETE FROM relationships
WHERE user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationshipDeleteVerified :exec
DELETE FROM relationships
WHERE user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid)
    AND (variant != 3 -- 3 = Blocked
        OR actor_id = @actor_id::uuid);

-- name: RelationshipPerspectiveGet :one
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id::uuid
    AND peer_id = @peer_id::uuid;

-- name: RelationshipPerspectiveGetByChannelID :one
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id::uuid
    AND channel_id = @channel_id::uuid;

-- name: RelationshipPerspectivesList :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id::uuid
    AND (sqlc.narg('filter_variant')::smallint IS NULL
        OR variant = sqlc.narg('filter_variant'))
ORDER BY
    updated_at DESC;

-- name: RelationshipHasBlockBetweenUserAndPeers :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            relationships r
        WHERE
            r.variant = 3 -- Blocked
            AND (
                -- Case A: Primary user is user1_id, peer is user2_id
(r.user1_id = @user_id::uuid
                    AND r.user2_id = ANY (@peer_ids::uuid[]))
                OR
                -- Case B: Peer is user1_id, Primary user is user2_id
(r.user2_id = @user_id::uuid
                    AND r.user1_id = ANY (@peer_ids::uuid[]))));

