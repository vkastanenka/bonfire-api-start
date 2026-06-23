-- ==========================================
-- META
-- ==========================================
-- name: UserProfileCount :one
SELECT
    COUNT(*)
FROM
    user_profiles;

-- ==========================================
-- CREATE
-- ==========================================
-- name: UserProfileCreate :one
INSERT INTO user_profiles(user_id, display_name)
    VALUES ($1, $2)
RETURNING
    *;

-- ==========================================
-- GET
-- ==========================================
-- name: UserProfileGetByUserID :one
SELECT
    *
FROM
    user_profiles
WHERE
    user_id = $1
LIMIT 1;

-- TODO: Not just 1, need to allow many since display name is not unique
-- name: UserProfileGetByDisplayName :one
SELECT
    *
FROM
    user_profiles
WHERE
    lower(display_name) = lower($1)
LIMIT 1;

-- ==========================================
-- UPDATE
-- ==========================================
-- name: UserProfileUpdateDisplayName :one
UPDATE
    user_profiles
SET
    display_name = $2
WHERE
    user_id = $1
RETURNING
    *;

-- ==========================================
-- DELETE
-- ==========================================
-- name: UserProfileDeleteByUserID :exec
DELETE FROM user_profiles
WHERE user_id = $1;

