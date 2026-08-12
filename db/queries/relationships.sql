-- name: RelationshipDeleteByUser :exec
DELETE FROM relationships
WHERE user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid)
    AND (type != 3
        OR actor_id = @actor_id::uuid);

-- name: RelationshipGet :one
SELECT
    relationships.*
FROM
    relationships
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationshipGetByChannel :one
SELECT
    relationships.*
FROM
    relationships
WHERE
    channel_id = @channel_id::uuid;

-- name: RelationshipListTypeByUser :many
SELECT
    CASE WHEN user1_id = @user_id::uuid THEN
        user2_id
    ELSE
        user1_id
    END AS peer_id,
    actor_id,
    channel_id,
    created_at,
    updated_at,
    type
FROM
    relationships
WHERE (user1_id = @user_id::uuid
    OR user2_id = @user_id::uuid)
AND type = @type_val::smallint
ORDER BY
    created_at DESC
LIMIT @batch_limit::int;

-- name: RelationshipSave :one
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
        relationships.*;

