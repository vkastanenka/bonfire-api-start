-- name: FindSharedDMChannel :one
-- Checks to see if an active type 1 (DM) channel exists between two specified users
SELECT
    c.id
FROM
    channels c
    JOIN channel_members cm1 ON c.id = cm1.channel_id
    JOIN channel_members cm2 ON c.id = cm2.channel_id
WHERE
    c.type = 1
    AND cm1.user_id = $1
    AND cm2.user_id = $2
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

