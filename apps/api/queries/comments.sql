-- name: CreateIssueComment :one
INSERT INTO issue_comments (issue_id, author_id, body)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetIssueComment :one
SELECT c.id, c.issue_id, c.author_id, u.name AS author_name, c.body, c.created_at, c.updated_at
FROM issue_comments c
JOIN users u ON u.id = c.author_id
WHERE c.issue_id = $1 AND c.id = $2;

-- name: ListIssueComments :many
SELECT c.id, c.issue_id, c.author_id, u.name AS author_name, c.body, c.created_at, c.updated_at
FROM issue_comments c
JOIN users u ON u.id = c.author_id
WHERE c.issue_id = $1
ORDER BY c.created_at ASC, c.id ASC;

-- name: UpdateIssueComment :exec
UPDATE issue_comments
SET body = $3, updated_at = now()
WHERE issue_id = $1 AND id = $2;

-- name: DeleteIssueComment :exec
DELETE FROM issue_comments
WHERE issue_id = $1 AND id = $2;
