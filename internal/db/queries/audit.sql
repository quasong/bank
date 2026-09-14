-- name: InsertAuditLog :exec
INSERT INTO audit_logs (id, actor_id, action, ip, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5, $6);
