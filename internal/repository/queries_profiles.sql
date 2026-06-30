-- ==========================================
-- META
-- ==========================================
-- name: ProfileCount :one
SELECT
    COUNT(*)
FROM
    profiles;

-- ==========================================
-- CREATE
-- ==========================================
-- name: ProfileCreate :one
INSERT INTO profiles(user_id, display_name)
    VALUES ($1, $2)
RETURNING
    *;

-- ==========================================
-- GET
-- ==========================================
-- name: ProfileGetByUserID :one
SELECT
    *
FROM
    profiles
WHERE
    user_id = $1
LIMIT 1;

-- TODO: Not just 1, need to allow many since display name is not unique
-- name: ProfileGetByDisplayName :one
SELECT
    *
FROM
    profiles
WHERE
    lower(display_name) = lower($1)
LIMIT 1;

-- ==========================================
-- UPDATE
-- ==========================================
-- name: ProfileUpdateDisplayName :one
UPDATE
    profiles
SET
    display_name = $2
WHERE
    user_id = $1
RETURNING
    *;

-- ==========================================
-- DELETE
-- ==========================================
-- name: ProfileDeleteByUserID :exec
DELETE FROM profiles
WHERE user_id = $1;

