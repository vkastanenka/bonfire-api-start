-- name: AttachmentCreateBatch :exec
INSERT INTO message_attachments(id, message_id, file_name, file_size, content_type, url, width, height, created_at)
SELECT
    unnest(@ids::uuid[]),
    @message_id,
    unnest(@file_names::text[]),
    unnest(@file_sizes::int[]),
    unnest(@content_types::text[]),
    unnest(@urls::text[]),
    unnest(@widths::int[]),
    unnest(@heights::int[]),
    unnest(@created_ats::timestamptz[])
WHERE
    @message_id IS NOT NULL;

-- name: AttachmentDeleteByMessage :exec
DELETE FROM message_attachments
WHERE message_id = $1;

