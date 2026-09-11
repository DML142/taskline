-- name: CreateIssue :one
INSERT INTO issues (project_id, title, description, status, priority, creator_id, assignee_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at;

-- name: GetIssue :one
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1 AND id = $2;

-- name: ListIssuesForProject :many
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1
ORDER BY created_at DESC, id DESC;

-- name: ListIssuesForProjectByStatus :many
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1 AND status = $2
ORDER BY created_at DESC, id DESC;

-- name: ListIssuesForProjectByAssignee :many
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1 AND assignee_id = $2
ORDER BY created_at DESC, id DESC;

-- name: ListIssuesForProjectByStatusAndAssignee :many
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1 AND status = $2 AND assignee_id = $3
ORDER BY created_at DESC, id DESC;

-- name: ListIssuesForProjectByFilters :many
SELECT id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at
FROM issues
WHERE project_id = $1
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status))
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR assignee_id = sqlc.narg(assignee_id))
  AND (sqlc.arg(priority)::text = '' OR priority = sqlc.arg(priority))
ORDER BY created_at DESC, id DESC;

-- name: UpdateIssue :one
UPDATE issues
SET title = $3,
    description = $4,
    status = $5,
    priority = $6,
    assignee_id = $7,
    updated_at = now()
WHERE project_id = $1 AND id = $2
RETURNING id, project_id, title, description, status, priority, creator_id, assignee_id, created_at, updated_at;
