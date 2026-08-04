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

