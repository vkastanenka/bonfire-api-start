-- name: FindSharedDMChannel :one
-- Checks to see if an active type 1 (DM) channel exists between two specified users
SELECT
    cm1.channel_id
FROM
    channel_members cm1
    JOIN channel_members cm2 ON cm1.channel_id = cm2.channel_id
    JOIN channels c ON cm1.channel_id = c.id
WHERE
    c.type = 1
    AND cm1.user_id = $1
    AND cm2.user_id = $2
    AND cm1.user_id != cm2.user_id
LIMIT 1;

-- name: CreateChannel :one
-- Spins up the foundation shell container for a message feed
INSERT INTO channels(type, name, guild_id)
    VALUES ($1, $2, $3)
RETURNING
    *;

-- name: AddChannelMember :exec
-- Binds a distinct user identity to a targeted channel access group
INSERT INTO channel_members(channel_id, user_id)
    VALUES ($1, $2);

-- name: ChannelGetByID :one
SELECT
    *
FROM
    channels
WHERE
    id = $1
LIMIT 1;

