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
    u.id AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON u.id = r.user2_id
WHERE
    r.user1_id = @user_id
    AND r.status = @status
UNION ALL
SELECT
    r.status,
    r.action_user_id,
    r.created_at,
    u.id AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON u.id = r.user1_id
WHERE
    r.user2_id = @user_id
    AND r.status = @status;

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

