-- name: AttachmentCreateBatch :exec
WITH input_data AS (
    SELECT
        *
    FROM
        jsonb_populate_recordset(NULL::message_attachments, @attachments_json::jsonb))
INSERT INTO message_attachments(id, message_id, file_name, file_size, content_type, url, width, height, created_at)
SELECT
    id,
    COALESCE(message_id, @message_id::uuid),
    file_name,
    file_size,
    content_type,
    url,
    width,
    height,
    created_at
FROM
    input_data;

-- name: AttachmentDelete :exec
DELETE FROM message_attachments
WHERE id = @attachment_id::uuid;

