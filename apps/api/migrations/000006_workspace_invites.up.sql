ALTER TABLE workspace_members
ADD COLUMN added_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL;

CREATE INDEX workspace_members_workspace_adder_idx
ON workspace_members (workspace_id, added_by_user_id);

CREATE TABLE workspace_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    email TEXT NOT NULL CHECK (email = lower(btrim(email))),
    role workspace_role NOT NULL CHECK (role IN ('ADMIN', 'MEMBER', 'VIEWER')),
    token_hash BYTEA NOT NULL UNIQUE,
    created_by_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX workspace_invites_one_open_email_idx
ON workspace_invites (workspace_id, email)
WHERE accepted_at IS NULL AND revoked_at IS NULL;

CREATE INDEX workspace_invites_workspace_created_idx
ON workspace_invites (workspace_id, created_at DESC);
