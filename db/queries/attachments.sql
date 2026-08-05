-- name: AttachmentMemberCreateBatch :exec
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
        created_at)
WHERE
    EXISTS (
        SELECT
            1
        FROM
            messages m
            JOIN channel_members cm ON cm.channel_id = m.channel_id
        WHERE
            m.id = @message_id::uuid
            AND cm.user_id = @user_id::uuid);

-- name: AttachmentMemberDelete :exec
DELETE FROM message_attachments ma
WHERE ma.id = @attachment_id::uuid
    AND EXISTS (
        SELECT
            1
        FROM
            messages m
            JOIN channel_members cm ON cm.channel_id = m.channel_id
        WHERE
            m.id = ma.message_id
            AND cm.user_id = @user_id::uuid);

