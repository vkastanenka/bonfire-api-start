-- name: AttachmentCreateBatch :many
INSERT INTO message_attachments(id, message_id, created_at, file_size, width, height, file_name, content_type, url)
SELECT
    m.id,
    m.message_id,
    m.created_at,
    m.file_size,
    m.width,
    m.height,
    m.file_name,
    m.content_type,
    m.url
FROM
    UNNEST(@ids::uuid[], @message_ids::uuid[], @created_ats::timestamptz[], @file_sizes::bigint[], @widths::integer[], @heights::integer[], @file_names::text[], @content_types::text[], @urls::text[]) AS m(id,
        message_id,
        created_at,
        file_size,
        width,
        height,
        file_name,
        content_type,
        url)
RETURNING
    message_attachments.*;

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

