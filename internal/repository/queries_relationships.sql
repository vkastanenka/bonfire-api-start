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
-- name: RelationshipsListByUser :many
SELECT
    r.status,
    r.action_user_id,
    r.created_at,
(
        CASE WHEN r.user1_id = @user_id THEN
            r.user2_id
        ELSE
            r.user1_id
        END)::uuid AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON u.id = CASE WHEN r.user1_id = @user_id THEN
        r.user2_id
    ELSE
        r.user1_id
    END
WHERE (r.user1_id = @user_id
    OR r.user2_id = @user_id)
AND (r.status != 'blocked'::relationship_status
    OR r.action_user_id = @user_id);

-- name: RelationshipsListPendingByUser :many
SELECT
    r.status,
    r.action_user_id,
    r.created_at,
(
        CASE WHEN r.user1_id = @user_id THEN
            r.user2_id
        ELSE
            r.user1_id
        END)::uuid AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON u.id = CASE WHEN r.user1_id = @user_id THEN
        r.user2_id
    ELSE
        r.user1_id
    END
WHERE (r.user1_id = @user_id
    OR r.user2_id = @user_id)
AND r.status = 'pending';

-- name: RelationshipsListFriendsByUser :many
SELECT
    r.status,
    r.action_user_id,
    r.created_at,
(
        CASE WHEN r.user1_id = @user_id THEN
            r.user2_id
        ELSE
            r.user1_id
        END)::uuid AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON u.id = CASE WHEN r.user1_id = @user_id THEN
        r.user2_id
    ELSE
        r.user1_id
    END
WHERE (r.user1_id = @user_id
    OR r.user2_id = @user_id)
AND r.status = 'friends';

-- name: RelationshipsListBlockedByUser :many
SELECT
    r.status,
    r.action_user_id,
    r.created_at,
(
        CASE WHEN r.user1_id = @user_id THEN
            r.user2_id
        ELSE
            r.user1_id
        END)::uuid AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON u.id = CASE WHEN r.user1_id = @user_id THEN
        r.user2_id
    ELSE
        r.user1_id
    END
WHERE (r.user1_id = @user_id
    OR r.user2_id = @user_id)
AND r.status = 'blocked'
AND r.action_user_id = @user_id;

-- ==========================================
-- GET
-- ==========================================
-- name: RelationshipGet :one
SELECT
    user1_id,
    user2_id,
    action_user_id,
    status,
    created_at,
    updated_at
FROM
    relationships
WHERE
    user1_id = $1
    AND user2_id = $2;

-- ==========================================
-- UPDATE
-- ==========================================
-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, status, action_user_id)
    VALUES ($1, $2, $3, $4)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        status = EXCLUDED.status,
        action_user_id = EXCLUDED.action_user_id,
        updated_at = CURRENT_TIMESTAMP
    RETURNING
        user1_id,
        user2_id,
        action_user_id,
        status,
        created_at,
        updated_at;

-- ==========================================
-- DELETE
-- ==========================================
-- name: RelationshipDelete :exec
DELETE FROM relationships
WHERE user1_id = $1
    AND user2_id = $2;

