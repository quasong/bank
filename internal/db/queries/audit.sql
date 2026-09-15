-- name: InsertAuditLog :exec
INSERT INTO audit_logs (id, actor_id, action, ip, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditLogsByActor :many
SELECT id, actor_id, action, ip, user_agent, metadata, created_at
FROM audit_logs
WHERE actor_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;
