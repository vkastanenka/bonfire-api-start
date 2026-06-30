-- ==========================================
-- RELATIONSHIPS
-- ==========================================
-- name: RelationshipGet :one
SELECT
    *
FROM
    relationships
WHERE
    user1_id = $1
    AND user2_id = $2;

-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, status, action_user_id)
    VALUES ($1, $2, $3, $4)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        status = EXCLUDED.status,
        action_user_id = EXCLUDED.action_user_id,
        updated_at = CURRENT_TIMESTAMP
    RETURNING
        *;

-- name: RelationshipDelete :exec
DELETE FROM relationships
WHERE user1_id = $1
    AND user2_id = $2;

-- name: RelationshipsListByUser :many
-- This query fetches the relationship and joins the OTHER user's profile info.
SELECT
    r.status,
    r.action_user_id,
    r.created_at,
    u.id AS related_user_id,
    u.username,
    u.status AS user_status
FROM
    relationships r
    JOIN users u ON (u.id = r.user1_id
            OR u.id = r.user2_id)
        AND u.id != $1
WHERE (r.user1_id = $1
    OR r.user2_id = $1)
AND r.status = $2;

