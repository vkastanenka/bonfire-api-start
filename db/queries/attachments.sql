-- name: AttachmentCreateBatch :many
INSERT INTO message_attachments(id, message_id, created_at, file_size, width, height, file_name, content_type, url)
SELECT
    id,
    message_id,
    created_at,
    file_size,
    width,
    height,
    file_name,
    content_type,
    url
FROM
    jsonb_to_recordset(@payload::jsonb) AS x(id uuid,
        message_id uuid,
        created_at timestamptz,
        file_size bigint,
        width integer,
        height integer,
        file_name text,
        content_type text,
        url text)
RETURNING
    *;

-- name: AttachmentGetBatchByMessageIDs :many
SELECT
    *
FROM
    message_attachments
WHERE
    message_id = ANY (@message_ids::uuid[])
ORDER BY
    message_id,
    id ASC;

-- name: AttachmentDelete :exec
DELETE FROM message_attachments
WHERE id = @attachment_id::uuid;

