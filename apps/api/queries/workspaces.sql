-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug)
VALUES ($1, $2)
RETURNING id, name, slug, created_at, updated_at;

-- name: CreateWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role, added_by_user_id)
VALUES ($1, $2, $3, $4)
RETURNING workspace_id, user_id, role, added_by_user_id, created_at;

-- name: GetWorkspaceMember :one
SELECT workspace_id, user_id, role, added_by_user_id, created_at
FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: ListWorkspacesForUser :many
SELECT w.id, w.name, w.slug, w.created_at, w.updated_at, wm.role
FROM workspaces AS w
JOIN workspace_members AS wm ON wm.workspace_id = w.id
WHERE wm.user_id = $1
ORDER BY w.created_at, w.id;

-- name: GetWorkspaceForMember :one
SELECT w.id, w.name, w.slug, w.created_at, w.updated_at, wm.role
FROM workspaces AS w
JOIN workspace_members AS wm ON wm.workspace_id = w.id
WHERE w.id = $1 AND wm.user_id = $2;

-- name: ListWorkspaceMembers :many
SELECT u.id, u.email, u.name, wm.role, wm.added_by_user_id, adder.name AS added_by_name, wm.created_at
FROM workspace_members AS wm
JOIN users AS u ON u.id = wm.user_id
LEFT JOIN users AS adder ON adder.id = wm.added_by_user_id
WHERE wm.workspace_id = $1
ORDER BY wm.created_at, u.id;

-- name: FindWorkspaceUserByEmail :one
SELECT id, email, name, password_hash, created_at, updated_at
FROM users
WHERE email = $1;

-- name: UpdateWorkspaceName :one
UPDATE workspaces
SET name = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, slug, created_at, updated_at;

-- name: UpdateWorkspaceMemberRole :one
UPDATE workspace_members
SET role = $3
WHERE workspace_id = $1 AND user_id = $2
RETURNING workspace_id, user_id, role, added_by_user_id, created_at;

-- name: DeleteWorkspaceMember :execrows
DELETE FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: DeleteWorkspace :execrows
DELETE FROM workspaces
WHERE id = $1;
