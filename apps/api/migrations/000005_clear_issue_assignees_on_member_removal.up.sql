CREATE FUNCTION clear_issue_assignees_on_member_removal()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE issues
    SET assignee_id = NULL,
        updated_at = now()
    FROM projects
    WHERE issues.project_id = projects.id
      AND projects.workspace_id = OLD.workspace_id
      AND issues.assignee_id = OLD.user_id;
    RETURN OLD;
END;
$$;

CREATE TRIGGER workspace_member_issue_assignee_cleanup
BEFORE DELETE ON workspace_members
FOR EACH ROW
EXECUTE FUNCTION clear_issue_assignees_on_member_removal();
