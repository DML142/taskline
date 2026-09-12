CREATE TABLE issue_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES issues (id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users (id),
    body TEXT NOT NULL CHECK (char_length(btrim(body)) BETWEEN 1 AND 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX issue_comments_issue_created_idx ON issue_comments (issue_id, created_at, id);
