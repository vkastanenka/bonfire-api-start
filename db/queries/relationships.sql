-- name: RelationshipDeleteVerified :exec
DELETE FROM relationships
WHERE user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid)
    AND (type != 3
        OR actor_id = @actor_id::uuid);

-- name: RelationshipGet :one
SELECT
    user1_id,
    user2_id,
    actor_id,
    channel_id,
    created_at,
    updated_at,
    type
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationshipUpsert :one
INSERT INTO relationships(user1_id, user2_id, actor_id, channel_id, created_at, updated_at, type)
    VALUES (LEAST(@user1_id::uuid, @user2_id::uuid), GREATEST(@user1_id::uuid, @user2_id::uuid), @actor_id::uuid, sqlc.narg('channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        actor_id = EXCLUDED.actor_id,
        channel_id = CASE WHEN EXCLUDED.type = 2 THEN
            COALESCE(EXCLUDED.channel_id, relationships.channel_id)
        ELSE
            NULL
        END,
        updated_at = EXCLUDED.updated_at,
        type = EXCLUDED.type
    RETURNING
        user1_id,
        user2_id,
        actor_id,
        channel_id,
        created_at,
        updated_at,
        type;

-- name: RelationshipUserListAggregate :many
WITH target_ids AS (
    SELECT
        r.user1_id,
        r.user2_id,
        r.updated_at,
        CASE WHEN r.user1_id = @user_id::uuid THEN
            r.user2_id
        ELSE
            r.user1_id
        END AS peer_id
    FROM
        relationships r
    WHERE (r.user1_id = @user_id::uuid
        OR r.user2_id = @user_id::uuid)
    AND (sqlc.narg('filter_type')::smallint IS NULL
        OR r.type = sqlc.narg('filter_type')::smallint)
    AND (sqlc.narg('cursor_updated_at')::timestamptz IS NULL
        OR (r.updated_at,
            r.user1_id,
            r.user2_id) <(sqlc.narg('cursor_updated_at')::timestamptz,
            sqlc.narg('cursor_user1_id')::uuid,
            sqlc.narg('cursor_user2_id')::uuid))
ORDER BY
    r.updated_at DESC,
    r.user1_id DESC,
    r.user2_id DESC
LIMIT @limit_val::int
)
SELECT
    r.user1_id,
    r.user2_id,
    r.actor_id,
    r.channel_id,
    r.created_at,
    r.updated_at,
    r.type,
    ti.peer_id,
    u.username AS peer_username,
    up.display_name AS peer_display_name,
    up.avatar_url AS peer_avatar_url
FROM
    target_ids ti
    JOIN relationships r ON r.user1_id = ti.user1_id
        AND r.user2_id = ti.user2_id
    JOIN users u ON u.id = ti.peer_id
    LEFT JOIN user_profiles up ON up.user_id = ti.peer_id
ORDER BY
    ti.updated_at DESC,
    ti.user1_id DESC,
    ti.user2_id DESC;

-- name: RelationshipUserPeerBlock :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            relationships r
        WHERE
            r.type = 3 -- Blocked
            AND ((r.user1_id = @user_id::uuid
                    AND r.user2_id = ANY (@peer_ids::uuid[]))
                OR (r.user2_id = @user_id::uuid
                    AND r.user1_id = ANY (@peer_ids::uuid[]))));

