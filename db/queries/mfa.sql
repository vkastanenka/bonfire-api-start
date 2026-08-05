-- name: UserMFABackupCodeConsume :one
UPDATE
    user_mfa_backup_codes
SET
    used_at = @used_at::timestamptz
WHERE
    user_id = @user_id::uuid
    AND code_hash = @code_hash::text
    AND used_at IS NULL
RETURNING
    id,
    user_id,
    created_at,
    used_at;

-- name: UserMFABackupCodesCountActive :one
-- Used to prompt users to regenerate backup codes when low (e.g., < 2 remaining).
SELECT
    COUNT(*)::bigint AS active_count
FROM
    user_mfa_backup_codes
WHERE
    user_id = @user_id::uuid
    AND used_at IS NULL;

-- name: UserMFABackupCodesCreateBatch :copyfrom
INSERT INTO user_mfa_backup_codes(id, user_id, created_at, used_at, code_hash)
    VALUES ($1, $2, $3, $4, $5);

-- name: UserMFABackupCodesDeleteUnused :exec
-- Wipe old backup codes when generating a new set.
DELETE FROM user_mfa_backup_codes
WHERE user_id = @user_id::uuid
    AND used_at IS NULL;

-- name: UserMFABackupCodesList :many
-- List usage state of backup codes for settings audit UI (omits raw hash for security).
SELECT
    id,
    user_id,
    created_at,
    used_at
FROM
    user_mfa_backup_codes
WHERE
    user_id = @user_id::uuid
ORDER BY
    created_at ASC
LIMIT @limit_val::int;

-- name: UserMFADisable :exec
DELETE FROM user_mfa
WHERE user_id = @user_id::uuid;

-- name: UserMFAEnable :one
-- Step 2 of MFA Setup: Mark TOTP active after successful initial token validation.
UPDATE
    user_mfa
SET
    enabled_at = @enabled_at::timestamptz,
    last_used_step = @initial_step::bigint,
    updated_at = @updated_at::timestamptz
WHERE
    user_id = @user_id::uuid
    AND enabled_at IS NULL
RETURNING
    user_id,
    created_at,
    updated_at,
    enabled_at,
    last_used_step,
    secret;

-- name: UserMFAGet :one
SELECT
    user_id,
    created_at,
    updated_at,
    enabled_at,
    last_used_step,
    secret
FROM
    user_mfa
WHERE
    user_id = @user_id::uuid
LIMIT 1;

-- name: UserMFAUpdateLastUsedStep :one
-- Replay Attack Prevention: Atomically advance last_used_step during authentication.
UPDATE
    user_mfa
SET
    last_used_step = @new_step::bigint,
    updated_at = @updated_at::timestamptz
WHERE
    user_id = @user_id::uuid
    AND enabled_at IS NOT NULL
    AND last_used_step < @new_step::bigint
RETURNING
    user_id,
    last_used_step,
    updated_at;

-- name: UserMFAUpsertSecret :one
-- Step 1 of MFA Setup: Create or replace pending secret before verification.
INSERT INTO user_mfa(user_id, created_at, updated_at, enabled_at, last_used_step, secret)
    VALUES (@user_id::uuid, @created_at::timestamptz, @updated_at::timestamptz, NULL, 0, @secret::text)
ON CONFLICT (user_id)
    DO UPDATE SET
        secret = EXCLUDED.secret,
        updated_at = EXCLUDED.updated_at,
        enabled_at = NULL,
        last_used_step = 0
    RETURNING
        user_id,
        created_at,
        updated_at,
        enabled_at,
        last_used_step,
        secret;

