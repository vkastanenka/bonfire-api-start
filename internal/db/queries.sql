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

-- name: UserCreateAggregate :exec
WITH new_user AS (
INSERT INTO users(id, email, username, password_hash, preferred_presence, verified_at, created_at, updated_at)
        VALUES (@user_id, @email, @username, @password_hash, @preferred_presence, @verified_at, @user_created_at, @user_updated_at))
    INSERT INTO user_profiles(user_id, display_name, avatar_url, created_at, updated_at)
        VALUES (@user_id, @display_name, @avatar_url, @profile_created_at, @profile_updated_at);

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

-- name: UserSave :one
UPDATE
    users
SET
    password_hash = $2,
    preferred_presence = $3,
    verified_at = $4,
    updated_at = $5
WHERE
    id = $1
RETURNING
    *;

-- name: UserProfileSave :one
INSERT INTO user_profiles(user_id, display_name, avatar_url, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5)
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
INSERT INTO outbox_events(event_type, payload)
    VALUES ($1, $2)
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
WHERE ($1::uuid IS NULL
    OR id < $1)
ORDER BY
    id DESC
LIMIT $2;

-- name: OutboxEventAcquireBatch :many
UPDATE
    outbox_events
SET
    locked_by = $2,
    lease_expires_at = CURRENT_TIMESTAMP +($3::text || ' seconds')::interval
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
        LIMIT $1
        FOR UPDATE
            SKIP LOCKED)
RETURNING
    *;

-- name: OutboxEventMarkProcessed :one
UPDATE
    outbox_events
SET
    processed_at = CURRENT_TIMESTAMP,
    locked_by = NULL,
    lease_expires_at = NULL
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventRecordFailure :one
UPDATE
    outbox_events
SET
    attempts = attempts + 1,
    last_error = $2,
    locked_by = NULL,
    lease_expires_at = NULL,
    next_attempt_at = CURRENT_TIMESTAMP +(INTERVAL '1 minute' * POWER(2, attempts + 1)::int)
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventMarkDeadLetter :one
UPDATE
    outbox_events
SET
    last_error = COALESCE($2, 'Manually marked dead letter by operator.'),
    attempts = max_attempts,
    locked_by = NULL,
    lease_expires_at = NULL
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventResetAttempts :one
UPDATE
    outbox_events
SET
    attempts = 0,
    next_attempt_at = CURRENT_TIMESTAMP,
    last_error = NULL,
    locked_by = NULL,
    lease_expires_at = NULL
WHERE
    id = $1
RETURNING
    *;

-- name: OutboxEventDelete :exec
DELETE FROM outbox_events
WHERE id = $1;

-- name: OutboxEventPurgeProcessed :exec
DELETE FROM outbox_events
WHERE processed_at <(CURRENT_TIMESTAMP - INTERVAL '7 days');

-- name: RelationshipsListByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = $1;

-- name: RelationshipsListPendingByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = $1
    AND type = 1;

-- name: RelationshipsListFriendsByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = $1
    AND type = 2;

-- name: RelationshipsListBlockedByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = $1
    AND type = 3
    AND actor_id = $1;

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

-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, type, actor_id)
    VALUES (LEAST(@user1_id::uuid, @user2_id::uuid), GREATEST(@user1_id::uuid, @user2_id::uuid), @type, @actor_id)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET type = EXCLUDED.type, actor_id = EXCLUDED.actor_id, updated_at = CURRENT_TIMESTAMP
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
    AND (type != 3 -- 3 = Blocked
        OR actor_id = @actor_id::uuid);

