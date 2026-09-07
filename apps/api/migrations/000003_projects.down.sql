DROP TABLE projects;
ALTER TABLE workspaces DROP CONSTRAINT workspaces_slug_format_check;
ALTER TABLE workspaces DROP CONSTRAINT workspaces_slug_key;
ALTER TABLE workspaces DROP COLUMN slug;
