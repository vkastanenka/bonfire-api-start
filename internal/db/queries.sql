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

-- name: ChannelGetForMemberUpdate :one
SELECT
    c.*
FROM
    channels c
    INNER JOIN channel_members cm ON cm.channel_id = c.id
WHERE
    c.id = @channel_id::uuid
    AND cm.user_id = @user_id::uuid
FOR UPDATE
    OF c;

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
    cm.channel_id,
    cm.user_id,
    rp.peer_id AS peer_user_id,
    c.type AS channel_type,
    COALESCE(c.name, rp.display_name, rp.username) AS channel_name,
    COALESCE(c.icon_url, rp.avatar_url) AS channel_icon_url,
    c.last_message_id AS channel_last_message_id,
    cm.last_read_message_id,
    cm.mention_count,
    cm.created_at,
    cm.last_read_at,
    cm.pinned_at AS member_pinned_at,
    cm.dm_visibility,
    c.updated_at AS channel_updated_at
FROM
    channel_members cm
    JOIN channels c ON cm.channel_id = c.id
    LEFT JOIN relationship_perspectives rp ON c.type = 0
        AND rp.user_id = cm.user_id
        AND rp.channel_id = c.id
WHERE
    cm.user_id = @user_id::uuid
    AND cm.dm_visibility = 1 -- 1: VISIBLE
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
INSERT INTO channel_members(channel_id, user_id, created_at, updated_at, last_read_at, mention_count, last_read_message_id, dm_visibility, pinned_at)
SELECT
    @channel_id::uuid,
    u.user_id,
    u.created_at,
    u.updated_at,
    u.last_read_at,
    u.mention_count,
    u.last_read_message_id,
    u.dm_visibility,
    u.pinned_at
FROM
    ROWS
FROM (unnest(@user_ids::uuid[]),
    unnest(@created_ats::timestamptz[]),
    unnest(@updated_ats::timestamptz[]),
    unnest(@last_read_ats::timestamptz[]),
    unnest(@mention_counts::int[]),
    unnest(@last_read_message_ids::uuid[]),
    unnest(@dm_visibilities::smallint[]),
    unnest(@pinned_ats::timestamptz[])) AS u(user_id,
        created_at,
        updated_at,
        last_read_at,
        mention_count,
        last_read_message_id,
        dm_visibility,
        pinned_at)
ON CONFLICT (channel_id,
    user_id)
    DO UPDATE SET
        updated_at = EXCLUDED.updated_at,
        last_read_at = EXCLUDED.last_read_at,
        mention_count = EXCLUDED.mention_count,
        last_read_message_id = EXCLUDED.last_read_message_id,
        dm_visibility = EXCLUDED.dm_visibility,
        pinned_at = EXCLUDED.pinned_at;

-- name: ChannelMemberCount :one
SELECT
    COUNT(*)::int AS count
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid;

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
    channel_id,
    user_id,
    created_at,
    updated_at,
    last_read_at,
    mention_count,
    last_read_message_id,
    pinned_at,
    dm_visibility
FROM
    channel_members
WHERE
    channel_id = @channel_id::uuid
ORDER BY
    created_at ASC,
    user_id ASC;

-- name: ChannelMemberListItemsByChannel :many
SELECT
    cm.channel_id,
    cm.user_id,
    cm.created_at AS member_since,
    cm.last_read_at,
    ua.username,
    ua.display_name,
    ua.avatar_url,
    ua.created_at AS user_created_at
FROM
    channel_members cm
    JOIN user_aggregates ua ON ua.id = cm.user_id
WHERE
    cm.channel_id = @channel_id::uuid
    AND (@cursor_created_at::timestamptz IS NULL
        OR (cm.created_at,
            cm.user_id) >(@cursor_created_at::timestamptz,
            @cursor_user_id::uuid))
ORDER BY
    cm.created_at ASC,
    cm.user_id ASC
LIMIT @limit_val::int;

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

-- name: ChannelMemberTogglePinned :exec
UPDATE
    channel_members
SET
    pinned_at = CASE WHEN pinned_at IS NULL THEN
        @pinned_at::timestamptz
    ELSE
        NULL
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberUpdateRead :exec
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

-- name: ChannelMemberCloseDM :exec
-- Hides or closes the DM channel for a specific member
UPDATE
    channel_members
SET
    dm_visibility = 0, -- 0: HIDDEN
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- name: ChannelMemberOpenDM :exec
-- Unhides/reopens the DM channel
UPDATE
    channel_members
SET
    dm_visibility = 1, -- 1: VISIBLE
    updated_at = @updated_at::timestamptz
WHERE
    channel_id = @channel_id::uuid
    AND user_id = @user_id::uuid;

-- ============================================================================
-- MESSAGES
-- ============================================================================
-- name: MessageCreate :one
INSERT INTO messages(id, channel_id, reply_to_message_id, author_id, created_at, updated_at, edited_at, pinned_at, content)
    VALUES (@id::uuid, @channel_id::uuid, sqlc.narg('reply_to_message_id')::uuid, sqlc.narg('author_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, sqlc.narg('edited_at')::timestamptz, sqlc.narg('pinned_at')::timestamptz, sqlc.narg('content')::text)
RETURNING
    *;

-- name: MessageDelete :exec
DELETE FROM messages
WHERE id = @id::uuid;

-- name: MessageGet :one
SELECT
    id,
    channel_id,
    reply_to_message_id,
    author_id,
    created_at,
    updated_at,
    edited_at,
    pinned_at,
    content
FROM
    messages
WHERE
    id = @id::uuid;

-- name: MessageGetAggregate :one
SELECT
    mb.id,
    mb.channel_id,
    mb.reply_to_message_id,
    mb.author_id,
    mb.author_username,
    mb.author_display_name,
    mb.author_avatar_url,
    mb.created_at,
    mb.updated_at,
    mb.edited_at,
    mb.pinned_at,
    mb.content,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    message_base_aggregates mb
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
            ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
        FROM
            message_attachments a
        WHERE
            a.message_id = mb.id) att ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
            ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
        FROM
            message_reactions r
        WHERE
            r.message_id = mb.id) rec ON TRUE
WHERE
    mb.id = @id::uuid;

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

-- name: MessageListAggregateAfter :many
-- Fetches newer messages strictly after the cursor tuple (Chronological ASC)
WITH hydrated_messages AS (
    SELECT
        mb.id,
        mb.channel_id,
        mb.reply_to_message_id,
        mb.author_id,
        mb.author_username,
        mb.author_display_name,
        mb.author_avatar_url,
        mb.created_at,
        mb.updated_at,
        mb.edited_at,
        mb.pinned_at,
        mb.content,
        COALESCE(att.attachments, '[]'::json) AS attachments,
        COALESCE(rec.reactions, '[]'::json) AS reactions
    FROM
        message_base_aggregates mb
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
                ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
            FROM
                message_attachments a
            WHERE
                a.message_id = mb.id) att ON TRUE
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
                ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
            FROM
                message_reactions r
            WHERE
                r.message_id = mb.id) rec ON TRUE
        WHERE
            mb.channel_id = @channel_id::uuid
)
    SELECT
        hm.*
    FROM
        hydrated_messages hm
    WHERE
        hm.channel_id = @channel_id::uuid
        AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
            OR (hm.created_at,
                hm.id) >(sqlc.narg('cursor_created_at')::timestamptz,
                sqlc.narg('cursor_id')::uuid))
    ORDER BY
        hm.created_at ASC,
        hm.id ASC
    LIMIT @limit_val::int;

-- name: MessageListAggregateAround :many
-- Fetches older/target messages and newer messages relative to target, returned ASC
WITH target AS (
    SELECT
        created_at
    FROM
        messages
    WHERE
        id = @target_id::uuid
),
older_window AS (
    SELECT
        m.id,
        m.created_at
    FROM
        messages m,
        target t
    WHERE
        m.channel_id = @channel_id::uuid
        AND (m.created_at,
            m.id) <=(t.created_at,
            @target_id::uuid)
    ORDER BY
        m.created_at DESC,
        m.id DESC
    LIMIT @older_limit::int
),
newer_window AS (
    SELECT
        m.id,
        m.created_at
    FROM
        messages m,
        target t
    WHERE
        m.channel_id = @channel_id::uuid
        AND (m.created_at,
            m.id) >(t.created_at,
            @target_id::uuid)
    ORDER BY
        m.created_at ASC,
        m.id ASC
    LIMIT @newer_limit::int
),
around_ids AS (
    SELECT
        id,
        created_at
    FROM
        older_window
    UNION ALL
    SELECT
        id,
        created_at
    FROM
        newer_window
)
SELECT
    mb.id,
    mb.channel_id,
    mb.reply_to_message_id,
    mb.author_id,
    mb.author_username,
    mb.author_display_name,
    mb.author_avatar_url,
    mb.created_at,
    mb.updated_at,
    mb.edited_at,
    mb.pinned_at,
    mb.content,
    COALESCE(att.attachments, '[]'::json) AS attachments,
    COALESCE(rec.reactions, '[]'::json) AS reactions
FROM
    around_ids ai
    JOIN message_base_aggregates mb ON mb.id = ai.id
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
            ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
        FROM
            message_attachments a
        WHERE
            a.message_id = mb.id) att ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
            ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
        FROM
            message_reactions r
        WHERE
            r.message_id = mb.id) rec ON TRUE
ORDER BY
    ai.created_at ASC,
    ai.id ASC;

-- name: MessageListAggregateBefore :many
-- Fetches older messages strictly before the cursor tuple (Reverse DESC order for indexing)
WITH hydrated_messages AS (
    SELECT
        mb.id,
        mb.channel_id,
        mb.reply_to_message_id,
        mb.author_id,
        mb.author_username,
        mb.author_display_name,
        mb.author_avatar_url,
        mb.created_at,
        mb.updated_at,
        mb.edited_at,
        mb.pinned_at,
        mb.content,
        COALESCE(att.attachments, '[]'::json) AS attachments,
        COALESCE(rec.reactions, '[]'::json) AS reactions
    FROM
        message_base_aggregates mb
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
                ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
            FROM
                message_attachments a
            WHERE
                a.message_id = mb.id) att ON TRUE
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
                ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
            FROM
                message_reactions r
            WHERE
                r.message_id = mb.id) rec ON TRUE
        WHERE
            mb.channel_id = @channel_id::uuid
)
    SELECT
        hm.*
    FROM
        hydrated_messages hm
    WHERE
        hm.channel_id = @channel_id::uuid
        AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
            OR (hm.created_at,
                hm.id) <(sqlc.narg('cursor_created_at')::timestamptz,
                sqlc.narg('cursor_id')::uuid))
    ORDER BY
        hm.created_at DESC,
        hm.id DESC
    LIMIT @limit_val::int;

-- name: MessageListPinnedAggregate :many
WITH hydrated_pinned_messages AS (
    SELECT
        mb.id,
        mb.channel_id,
        mb.reply_to_message_id,
        mb.author_id,
        mb.author_username,
        mb.author_display_name,
        mb.author_avatar_url,
        mb.created_at,
        mb.updated_at,
        mb.edited_at,
        mb.pinned_at,
        mb.content,
        COALESCE(att.attachments, '[]'::json) AS attachments,
        COALESCE(rec.reactions, '[]'::json) AS reactions
    FROM
        message_base_aggregates mb
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('id', a.id, 'file_name', a.file_name, 'file_size', a.file_size, 'content_type', a.content_type, 'url', a.url, 'width', a.width, 'height', a.height, 'created_at', a.created_at)
                ORDER BY a.created_at ASC) FILTER (WHERE a.id IS NOT NULL) AS attachments
            FROM
                message_attachments a
            WHERE
                a.message_id = mb.id) att ON TRUE
        LEFT JOIN LATERAL (
            SELECT
                json_agg(json_build_object('message_id', r.message_id, 'user_id', r.user_id, 'emoji', r.emoji, 'created_at', r.created_at)
                ORDER BY r.created_at ASC) FILTER (WHERE r.message_id IS NOT NULL) AS reactions
            FROM
                message_reactions r
            WHERE
                r.message_id = mb.id) rec ON TRUE
        WHERE
            mb.channel_id = @channel_id::uuid
            AND mb.pinned_at IS NOT NULL
            -- Keyset comparison using pinned_at:
            AND (@cursor_pinned_at::timestamptz IS NULL
                OR (mb.pinned_at,
                    mb.id) <(@cursor_pinned_at::timestamptz,
                    @cursor_id::uuid)))
    SELECT
        hm.*
    FROM
        hydrated_pinned_messages hm
    ORDER BY
        hm.pinned_at DESC,
        hm.id DESC
    LIMIT @limit_val::int;

-- name: MessageTogglePinned :one
UPDATE
    messages
SET
    pinned_at = CASE WHEN pinned_at IS NULL THEN
        @pinned_at::timestamptz
    ELSE
        NULL
    END,
    updated_at = @updated_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    *;

-- name: MessageUpdateContent :one
UPDATE
    messages
SET
    content = sqlc.narg('content')::text,
    updated_at = @updated_at::timestamptz,
    edited_at = @edited_at::timestamptz
WHERE
    id = @id::uuid
RETURNING
    *;

-- ============================================================================
-- MESSAGE ATTACHMENTS
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

-- name: AttachmentDeleteByMessage :exec
DELETE FROM message_attachments
WHERE message_id = $1;

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

-- ============================================================================
-- RELATIONSHIPS
-- ============================================================================
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

-- name: RelationshipGet :one
SELECT
    *
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationshipGetForUpdate :one
SELECT
    *
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid)
FOR UPDATE;

-- name: RelationshipGetByChannelID :one
SELECT
    *
FROM
    relationships
WHERE
    channel_id = @channel_id::uuid;

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

-- -- name: RelationshipPerspectivesList :many
-- SELECT
--     *
-- FROM
--     relationship_perspectives
-- WHERE
--     user_id = @user_id::uuid
--     AND (sqlc.narg('filter_variant')::smallint IS NULL
--         OR variant = sqlc.narg('filter_variant'))
-- ORDER BY
--     updated_at DESC;
-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, actor_id, channel_id, created_at, updated_at, variant)
    VALUES (LEAST(@user1_id::uuid, @user2_id::uuid), GREATEST(@user1_id::uuid, @user2_id::uuid), @actor_id, sqlc.narg('channel_id')::uuid, @created_at, @updated_at, @variant)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        actor_id = EXCLUDED.actor_id,
        channel_id = CASE WHEN EXCLUDED.variant = 2 THEN
            COALESCE(EXCLUDED.channel_id, relationships.channel_id)
        ELSE
            NULL
        END,
        updated_at = EXCLUDED.updated_at,
        variant = EXCLUDED.variant
    RETURNING
        *;

