-- name: CreateProject :one
INSERT INTO projects (workspace_id, name, slug, description)
VALUES ($1, $2, $3, $4)
RETURNING id, workspace_id, name, slug, description, archived, created_at, updated_at;

-- name: ListProjectsForWorkspace :many
SELECT id, workspace_id, name, slug, description, archived, created_at, updated_at
FROM projects
WHERE workspace_id = $1
ORDER BY archived, created_at, id;

-- name: GetProjectBySlug :one
SELECT id, workspace_id, name, slug, description, archived, created_at, updated_at
FROM projects
WHERE workspace_id = $1 AND slug = $2;

-- name: UpdateProject :one
UPDATE projects
SET name = $3, slug = $4, description = $5, archived = $6, updated_at = now()
WHERE workspace_id = $1 AND slug = $2
RETURNING id, workspace_id, name, slug, description, archived, created_at, updated_at;
