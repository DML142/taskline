ALTER TABLE workspaces ADD COLUMN slug TEXT;

UPDATE workspaces
SET slug = 'workspace-' || id::text
WHERE slug IS NULL;

ALTER TABLE workspaces ALTER COLUMN slug SET NOT NULL;
ALTER TABLE workspaces ADD CONSTRAINT workspaces_slug_key UNIQUE (slug);
ALTER TABLE workspaces ADD CONSTRAINT workspaces_slug_format_check
    CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$');

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    archived BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, slug)
);

CREATE INDEX projects_workspace_active_idx ON projects (workspace_id, created_at, id)
WHERE archived = false;
