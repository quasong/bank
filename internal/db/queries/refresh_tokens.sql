-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
    id, customer_id, token_hash, expires_at, ip, user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: GetRefreshTokenByHashForUpdate :one
SELECT id, customer_id, token_hash, expires_at, revoked_at, replaced_by, ip, user_agent, created_at
FROM refresh_tokens
WHERE token_hash = $1
FOR UPDATE;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = $2,
    replaced_by = $3
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeRefreshTokenByHash :execrows
UPDATE refresh_tokens
SET revoked_at = $2
WHERE token_hash = $1
  AND revoked_at IS NULL;
