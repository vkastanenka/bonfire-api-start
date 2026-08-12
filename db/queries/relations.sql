-- name: RelationDeleteByUser :exec
DELETE FROM relations
WHERE user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid)
    AND (type != 3
        OR actor_id = @actor_id::uuid);

-- name: RelationGet :one
SELECT
    relations.*
FROM
    relations
WHERE
    user1_id = LEAST(@user1_id::uuid, @user2_id::uuid)
    AND user2_id = GREATEST(@user1_id::uuid, @user2_id::uuid);

-- name: RelationGetByChannel :one
SELECT
    relations.*
FROM
    relations
WHERE
    channel_id = @channel_id::uuid;

-- name: RelationListTypeByUser :many
SELECT
    (
        CASE WHEN user1_id = @user_id::uuid THEN
            user2_id
        ELSE
            user1_id
        END)::uuid AS peer_id,
    actor_id,
    channel_id,
    created_at,
    updated_at,
    type
FROM
    relations
WHERE (user1_id = @user_id::uuid
    OR user2_id = @user_id::uuid)
AND type = @type_val::smallint
ORDER BY
    created_at DESC
LIMIT @batch_limit::int;

-- name: RelationSave :one
INSERT INTO relations(user1_id, user2_id, actor_id, channel_id, created_at, updated_at, type)
    VALUES (LEAST(@user1_id::uuid, @user2_id::uuid), GREATEST(@user1_id::uuid, @user2_id::uuid), @actor_id::uuid, sqlc.narg('channel_id')::uuid, @created_at::timestamptz, @updated_at::timestamptz, @type::smallint)
ON CONFLICT (user1_id, user2_id)
    DO UPDATE SET
        actor_id = EXCLUDED.actor_id,
        channel_id = CASE WHEN EXCLUDED.type = 2 THEN
            COALESCE(EXCLUDED.channel_id, relations.channel_id)
        ELSE
            NULL
        END,
        updated_at = EXCLUDED.updated_at,
        type = EXCLUDED.type
    RETURNING
        relations.*;

