-- name: UserDeleteRequestCreate :one
INSERT INTO user_delete_requests(user_id, scheduled_at)
    VALUES ($1, $2)
RETURNING
    *;

-- name: UserDeleteRequestGet :one
SELECT
    *
FROM
    user_delete_requests
WHERE
    user_id = $1
LIMIT 1;

-- name: UserDeleteRequestListDue :many
SELECT
    user_id
FROM
    user_delete_requests
WHERE
    scheduled_at <= CURRENT_TIMESTAMP
ORDER BY
    scheduled_at ASC;

-- name: UserDeleteRequestDelete :exec
DELETE FROM user_delete_requests
WHERE user_id = $1;

