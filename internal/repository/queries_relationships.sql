-- ==========================================
-- META
-- ==========================================
-- name: RelationshipsCount :one
SELECT
    COUNT(*)
FROM
    relationships;

-- ==========================================
-- LIST
-- ==========================================
-- name: RelationshipsListByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id
    AND (type != 2
        OR actor_id = @user_id);

-- name: RelationshipsListPendingByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id
    AND type = 0;

-- name: RelationshipsListFriendsByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id
    AND type = 1;

-- name: RelationshipsListBlockedByUserID :many
SELECT
    *
FROM
    relationship_perspectives
WHERE
    user_id = @user_id
    AND type = 2
    AND actor_id = @user_id;

-- ==========================================
-- GET
-- ==========================================
-- name: RelationshipGet :one
SELECT
    user1_id,
    user2_id,
    actor_id,
    type,
    created_at,
    updated_at
FROM
    relationships
WHERE
    user1_id = $1
    AND user2_id = $2;

-- name: RelationshipGetForUpdate :one
SELECT
    user1_id,
    user2_id,
    actor_id,
    type,
    created_at,
    updated_at
FROM
    relationships
WHERE
    user1_id = $1
    AND user2_id = $2
FOR UPDATE;

-- ==========================================
-- UPDATE
-- ==========================================
-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, type, actor_id)
    VALUES ($1, $2, $3, $4)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET type = EXCLUDED.type, actor_id = EXCLUDED.actor_id, updated_at = CURRENT_TIMESTAMP
RETURNING
    user1_id, user2_id, actor_id, type, created_at, updated_at;

-- ==========================================
-- DELETE
-- ==========================================
-- name: RelationshipDelete :exec
DELETE FROM relationships
WHERE user1_id = $1
    AND user2_id = $2;

-- name: RelationshipDeleteVerified :one
DELETE FROM relationships
WHERE user1_id = $1
    AND user2_id = $2
    AND (type != 2
        OR actor_id = $3)
RETURNING
    type;

