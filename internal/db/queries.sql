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

-- name: RelationshipGet :one
SELECT
    user1_id,
    user2_id,
    actor_id,
    variant,
    created_at,
    updated_at
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationshipGetForUpdate :one
SELECT
    user1_id,
    user2_id,
    actor_id,
    variant,
    created_at,
    updated_at
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid)
FOR UPDATE;

-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, actor_id, variant, created_at, updated_at)
    VALUES (LEAST(@user1_id::uuid, @user2_id::uuid), GREATEST(@user1_id::uuid, @user2_id::uuid), @actor_id, @variant, @created_at, @updated_at)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        variant = EXCLUDED.variant,
        actor_id = EXCLUDED.actor_id,
        updated_at = EXCLUDED.updated_at
    RETURNING
        user1_id,
        user2_id,
        actor_id,
        variant,
        created_at,
        updated_at;

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
    user_id,
    peer_id,
    variant,
    actor_id,
    is_initiator,
    created_at,
    updated_at,
    username,
    display_name,
    avatar_url,
    user_preferred_presence,
    channel_id
FROM
    relationship_perspectives
WHERE
    user_id = @user_id::uuid
    AND peer_id = @peer_id::uuid;

-- name: RelationshipPerspectivesList :many
SELECT
    user_id,
    peer_id,
    variant,
    actor_id,
    is_initiator,
    created_at,
    updated_at,
    username,
    display_name,
    avatar_url,
    user_preferred_presence,
    channel_id
FROM
    relationship_perspectives
WHERE
    user_id = @user_id::uuid
    AND (sqlc.narg('filter_variant')::smallint IS NULL
        OR variant = sqlc.narg('filter_variant'))
ORDER BY
    updated_at DESC;

-- ============================================================================
-- CHANNELS
-- ============================================================================
-- name: ChannelCreate :one
INSERT INTO channels(id, type, owner_id, name, icon_url, created_at, updated_at)
    VALUES (@id, @type, sqlc.narg('owner_id'), sqlc.narg('name'), sqlc.narg('icon_url'), @created_at, @updated_at)
RETURNING
    *;

-- name: ChannelGet :one
SELECT
    *
FROM
    channels
WHERE
    id = $1
LIMIT 1;

-- name: ChannelUpdate :one
UPDATE
    channels
SET
    name = sqlc.narg('name'),
    icon_url = sqlc.narg('icon_url'),
    owner_id = sqlc.narg('owner_id'),
    last_message_id = sqlc.narg('last_message_id'),
    updated_at = @updated_at
WHERE
    id = @id
RETURNING
    *;

-- name: ChannelUpdateLastMessage :exec
UPDATE
    channels
SET
    last_message_id = @last_message_id,
    updated_at = @updated_at
WHERE
    id = @channel_id;

-- name: ChannelDelete :exec
DELETE FROM channels
WHERE id = $1;

-- name: ChannelFindDM :one
SELECT
    c.*
FROM
    channels c
    JOIN dm_relationships dm ON c.id = dm.channel_id
WHERE
    dm.user1_id = @user1_id
    AND dm.user2_id = @user2_id;

-- name: ChannelListByUser :many
-- Used for populating the user's sidebar with channel info & peer profiles for DMs
SELECT
    cm.channel_id,
    cm.user_id,
    cm.last_read_message_id,
    cm.mention_count,
    cm.joined_at,
    c.type AS channel_type,
    c.owner_id AS channel_owner_id,
    -- Fall back to peer's display name/username for 1-on-1 DMs
    COALESCE(c.name, peer_up.display_name, peer_u.username) AS channel_name,
    COALESCE(c.icon_url, peer_up.avatar_url) AS channel_icon_url,
    c.last_message_id AS channel_last_message_id,
    c.updated_at AS channel_updated_at,
    peer_u.id AS peer_user_id,
    peer_u.preferred_presence AS peer_preferred_presence
FROM
    channel_members cm
    JOIN channels c ON cm.channel_id = c.id
    -- Join directly on channel_id for instant O(1) index lookup
    LEFT JOIN dm_relationships dm ON c.type = 0
        AND dm.channel_id = c.id
        -- Select whichever side of the pair IS NOT the current user
    LEFT JOIN users peer_u ON c.type = 0
        AND peer_u.id = CASE WHEN dm.user1_id = cm.user_id THEN
            dm.user2_id
        ELSE
            dm.user1_id
        END
    LEFT JOIN user_profiles peer_up ON peer_u.id = peer_up.user_id
WHERE
    cm.user_id = @user_id
ORDER BY
    c.updated_at DESC;

-- ============================================================================
-- CHANNEL MEMBERS
-- ============================================================================
-- name: ChannelMemberAdd :one
INSERT INTO channel_members(channel_id, user_id, joined_at, last_read_message_id, mention_count)
    VALUES (@channel_id, @user_id, @joined_at, sqlc.narg('last_read_message_id'), @mention_count)
ON CONFLICT (channel_id, user_id)
    DO UPDATE SET
        joined_at = EXCLUDED.joined_at
    RETURNING
        *;

-- name: ChannelMemberGet :one
SELECT
    *
FROM
    channel_members
WHERE
    channel_id = @channel_id
    AND user_id = @user_id
LIMIT 1;

-- name: ChannelMemberListByChannel :many
SELECT
    cm.channel_id,
    cm.user_id,
    cm.joined_at,
    cm.last_read_message_id,
    cm.mention_count,
    u.username,
    up.display_name,
    up.avatar_url,
    u.preferred_presence
FROM
    channel_members cm
    JOIN users u ON u.id = cm.user_id
    LEFT JOIN user_profiles up ON up.user_id = cm.user_id
WHERE
    cm.channel_id = @channel_id
ORDER BY
    cm.joined_at ASC;

-- name: ChannelMemberUpdateReadState :exec
UPDATE
    channel_members
SET
    last_read_message_id = @last_read_message_id,
    mention_count = 0
WHERE
    channel_id = @channel_id
    AND user_id = @user_id;

-- name: ChannelMemberIncrementMentionCount :exec
UPDATE
    channel_members
SET
    mention_count = mention_count + 1
WHERE
    channel_id = @channel_id
    AND user_id = @user_id;

-- name: ChannelMemberIncrementMentionCountBatch :exec
UPDATE
    channel_members
SET
    mention_count = mention_count + 1
WHERE
    channel_id = @channel_id
    AND user_id = ANY (@user_ids::uuid[]);

-- name: ChannelMemberRemove :exec
DELETE FROM channel_members
WHERE channel_id = @channel_id
    AND user_id = @user_id;

-- ============================================================================
-- MESSAGES
-- ============================================================================
-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, author_id, reply_to_message_id, content, is_pinned, created_at)
    VALUES (@id, @channel_id, sqlc.narg('author_id'), sqlc.narg('reply_to_message_id'), @content, @is_pinned, @created_at)
RETURNING
    *;

-- name: MessageGet :one
SELECT
    *
FROM
    messages
WHERE
    id = $1
LIMIT 1;

-- name: MessageListByChannelKeyset :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = @channel_id
    AND (@cursor_created_at::timestamptz IS NULL
        OR (created_at,
            id) <(@cursor_created_at::timestamptz,
            @cursor_id::uuid))
ORDER BY
    created_at DESC,
    id DESC
LIMIT @result_limit;

-- name: MessageListPinnedByChannel :many
SELECT
    *
FROM
    messages
WHERE
    channel_id = $1
    AND is_pinned = TRUE
ORDER BY
    created_at DESC;

-- name: MessageListReplies :many
-- Fetches direct replies to a specific parent message
SELECT
    m.*
FROM
    messages m
WHERE
    m.reply_to_message_id = @reply_to_message_id
ORDER BY
    m.id ASC;

-- name: MessageUpdateContent :one
UPDATE
    messages
SET
    content = @content,
    edited_at = @edited_at
WHERE
    id = @id
RETURNING
    *;

-- name: MessageSetPinned :exec
UPDATE
    messages
SET
    is_pinned = @is_pinned
WHERE
    id = @id;

-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = $1;

-- ============================================================================
-- MESSAGE REACTIONS
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

