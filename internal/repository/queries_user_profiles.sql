-- name: UserProfileCreate :one
INSERT INTO
    user_profiles (user_id, display_name)
VALUES
    ($ 1, $ 2) RETURNING *;

-- name: UserProfileGet :one
SELECT
    *
FROM
    user_profiles
WHERE
    user_id = $ 1
LIMIT
    1;

-- name: UserProfileGetByDisplayName :one
SELECT
    *
FROM
    user_profiles
WHERE
    lower(display_name) = lower($ 1)
LIMIT
    1;

-- name: UserProfileUpdateDisplayName :exec
UPDATE
    user_profiles
SET
    display_name = $ 2,
    updated_at = CURRENT_TIMESTAMP
WHERE
    user_id = $ 1;

-- name: UserProfileDelete :exec
DELETE FROM
    user_profiles
WHERE
    user_id = $ 1;

-- name: UserProfileCheckDisplayNameAvailability :one
SELECT
    NOT EXISTS (
        SELECT
            1
        FROM
            user_profiles
        WHERE
            lower(display_name) = lower($ 1)
    ) AS available;