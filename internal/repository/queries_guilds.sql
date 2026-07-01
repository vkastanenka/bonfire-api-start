-- ==========================================
-- GUILDS
-- ==========================================
-- name: GuildCreate :one
INSERT INTO guilds(owner_id)
    VALUES ($1)
RETURNING
    *;

-- name: GuildProfileCreate :one
INSERT INTO guild_profiles(guild_id, name, icon_url, banner_hex, description, visibility)
    VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
    *;

-- name: GuildGetFull :one
SELECT
    g.id,
    g.owner_id,
    g.created_at,
    gp.name,
    gp.icon_url,
    gp.banner_hex,
    gp.description,
    gp.visibility
FROM
    guilds g
    JOIN guild_profiles gp ON g.id = gp.guild_id
WHERE
    g.id = $1
LIMIT 1;

-- name: GuildDelete :exec
DELETE FROM guilds
WHERE id = $1;

-- ==========================================
-- GUILD MEMBERS
-- ==========================================
-- name: GuildMemberAdd :exec
INSERT INTO guild_members(guild_id, user_id)
    VALUES ($1, $2);

-- name: GuildMemberRemove :exec
DELETE FROM guild_members
WHERE guild_id = $1
    AND user_id = $2;

-- name: GuildListMembers :many
SELECT
    u.id,
    u.username
FROM
    users u
    JOIN guild_members gm ON u.id = gm.user_id
WHERE
    gm.guild_id = $1;

-- ==========================================
-- ROLES & PERMISSIONS
-- ==========================================
-- name: RoleCreate :one
INSERT INTO guild_roles(guild_id, name, color_hex, permissions)
    VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- name: RoleUpdate :one
UPDATE
    guild_roles
SET
    name = $2,
    color_hex = $3,
    permissions = $4
WHERE
    id = $1
    AND guild_id = $5
RETURNING
    *;

-- name: RoleDelete :exec
DELETE FROM guild_roles
WHERE id = $1
    AND guild_id = $2;

-- name: GuildListRoles :many
SELECT
    *
FROM
    guild_roles
WHERE
    guild_id = $1;

-- name: MemberAssignRole :exec
INSERT INTO guild_member_roles(guild_id, user_id, role_id)
    VALUES ($1, $2, $3);

-- name: MemberRemoveRole :exec
DELETE FROM guild_member_roles
WHERE guild_id = $1
    AND user_id = $2
    AND role_id = $3;

-- name: GetEffectivePermissions :one
SELECT
    COALESCE(BIT_OR(r.permissions), 0)::bigint AS effective_permissions
FROM
    guild_member_roles gmr
    JOIN guild_roles r ON gmr.role_id = r.id
WHERE
    gmr.guild_id = $1
    AND gmr.user_id = $2;

