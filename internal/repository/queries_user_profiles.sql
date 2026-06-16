-- user_profiles
-- name: CreateUserProfile :one
INSERT INTO
    user_profiles (user_id, display_name)
VALUES
    ($ 1, $ 2) RETURNING user_id,
    created_at,
    display_name;

-- name: GetUserProfile :one
SELECT
    user_id,
    created_at,
    updated_at,
    display_name
FROM
    user_profiles
WHERE
    user_id = $ 1;