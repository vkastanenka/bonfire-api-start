-- name: AttachmentCreateBatch :exec
INSERT INTO message_attachments(id, message_id, file_name, file_size, content_type, url, width, height, created_at)
SELECT
    u.id,
    @message_id::uuid,
    u.file_name,
    u.file_size,
    u.content_type,
    u.url,
    u.width,
    u.height,
    u.created_at
FROM
    unnest(@ids::uuid[], @file_names::text[], @file_sizes::bigint[], @content_types::text[], @urls::text[], @widths::int[], @heights::int[], @created_ats::timestamptz[]) AS u(id,
        file_name,
        file_size,
        content_type,
        url,
        width,
        height,
        created_at);

-- name: AttachmentDelete :exec
DELETE FROM message_attachments
WHERE id = @attachment_id::uuid;

