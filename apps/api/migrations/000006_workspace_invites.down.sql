DROP TABLE workspace_invites;
DROP INDEX workspace_members_workspace_adder_idx;
ALTER TABLE workspace_members DROP COLUMN added_by_user_id;
