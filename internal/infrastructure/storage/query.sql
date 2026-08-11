-- name: SaveUser :exec
INSERT INTO users (id, invited_by, created_at)
VALUES (sqlc.arg(id), sqlc.narg(invited_by), sqlc.arg(created_at));

-- name: ListDescendantUserIDs :many
WITH RECURSIVE compromised_chain AS (
    SELECT users.id FROM users WHERE users.id = sqlc.arg(root)
    UNION ALL
    SELECT u.id FROM users u
    JOIN compromised_chain cc ON u.invited_by = cc.id
)
SELECT cc.id FROM compromised_chain cc;

-- name: FindCredentialByExternalID :one
SELECT * FROM credentials WHERE external_id = sqlc.arg(external_id);

-- name: ListCredentialsByUserID :many
SELECT * FROM credentials WHERE user_id = sqlc.arg(user_id) ORDER BY created_at;

-- name: SaveCredential :exec
INSERT INTO credentials (id, external_id, user_id, public_key, sign_count, transports, created_at, revoked_at)
VALUES (sqlc.arg(id), sqlc.arg(external_id), sqlc.arg(user_id), sqlc.arg(public_key),
        sqlc.arg(sign_count), sqlc.arg(transports), sqlc.arg(created_at), sqlc.narg(revoked_at))
ON CONFLICT (id) DO UPDATE
SET sign_count = EXCLUDED.sign_count,
    revoked_at = EXCLUDED.revoked_at;

-- name: CountActiveInvitesByUser :one
SELECT count(*) FROM invites
WHERE created_by = sqlc.arg(created_by)
  AND used_by IS NULL
  AND expires_at > sqlc.arg(now);

-- name: LockUserForInviteCreation :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(user_id)::text, 0));

-- name: FindInviteByCodeHash :one
SELECT * FROM invites WHERE code_hash = sqlc.arg(code_hash);

-- name: FindInviteByIDForUpdate :one
SELECT * FROM invites WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: SaveInvite :exec
INSERT INTO invites (id, created_by, code_hash, used_by, expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(created_by), sqlc.arg(code_hash), sqlc.narg(used_by),
        sqlc.arg(expires_at), sqlc.arg(created_at))
ON CONFLICT (id) DO UPDATE
SET used_by = EXCLUDED.used_by;

-- name: FindRefreshTokenByTokenHash :one
SELECT * FROM refresh_tokens WHERE token_hash = sqlc.arg(token_hash);

-- name: ListActiveRefreshTokensByUserIDs :many
SELECT * FROM refresh_tokens
WHERE user_id = ANY(sqlc.arg(user_ids)::uuid[])
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now);

-- name: SaveRefreshTokens :batchexec
INSERT INTO refresh_tokens (
    id, user_id, token_hash, device_info, ip_address, expires_at, revoked_at, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET revoked_at = EXCLUDED.revoked_at;

-- name: SaveWrappedKeys :copyfrom
INSERT INTO wrapped_keys (id, user_id, credential_id, scope, wrapped_dek, wrap_algorithm, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);