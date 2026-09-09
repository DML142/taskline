-- name: CreateWorkspaceInvite :one
INSERT INTO workspace_invites (workspace_id, email, role, token_hash, created_by_user_id, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, workspace_id, email, role, token_hash, created_by_user_id, expires_at, accepted_at, revoked_at, created_at;

-- name: GetWorkspaceInviteByTokenHashForUpdate :one
SELECT id, workspace_id, email, role, token_hash, created_by_user_id, expires_at, accepted_at, revoked_at, created_at
FROM workspace_invites
WHERE token_hash = $1
FOR UPDATE;

-- name: RevokeOpenWorkspaceInvitesForEmail :exec
UPDATE workspace_invites
SET revoked_at = now()
WHERE workspace_id = $1 AND email = $2 AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: AcceptWorkspaceInvite :execrows
UPDATE workspace_invites
SET accepted_at = now()
WHERE id = $1 AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now();

-- name: ListOpenWorkspaceInvites :many
SELECT wi.id, wi.workspace_id, wi.email, wi.role, wi.created_by_user_id, creator.name AS created_by_name, wi.expires_at, wi.created_at
FROM workspace_invites AS wi
JOIN users AS creator ON creator.id = wi.created_by_user_id
WHERE wi.workspace_id = $1 AND wi.accepted_at IS NULL AND wi.revoked_at IS NULL AND wi.expires_at > now()
ORDER BY wi.created_at DESC, wi.id DESC;

-- name: ListOpenWorkspaceInvitesByCreator :many
SELECT wi.id, wi.workspace_id, wi.email, wi.role, wi.created_by_user_id, creator.name AS created_by_name, wi.expires_at, wi.created_at
FROM workspace_invites AS wi
JOIN users AS creator ON creator.id = wi.created_by_user_id
WHERE wi.workspace_id = $1 AND wi.created_by_user_id = $2 AND wi.accepted_at IS NULL AND wi.revoked_at IS NULL AND wi.expires_at > now()
ORDER BY wi.created_at DESC, wi.id DESC;

-- name: RevokeWorkspaceInvite :execrows
UPDATE workspace_invites
SET revoked_at = now()
WHERE id = $1 AND workspace_id = $2 AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: RevokeWorkspaceInviteByCreator :execrows
UPDATE workspace_invites
SET revoked_at = now()
WHERE id = $1 AND workspace_id = $2 AND created_by_user_id = $3 AND accepted_at IS NULL AND revoked_at IS NULL;
