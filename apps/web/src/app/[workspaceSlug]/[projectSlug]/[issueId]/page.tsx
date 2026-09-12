"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { CommentApi, type IssueComment } from "@/lib/comment-api";
import {
  IssueApi,
  type Issue,
  type IssuePriority,
  type IssueStatus,
} from "@/lib/issue-api";
import { ProjectApi } from "@/lib/project-api";
import {
  WorkspaceApi,
  type WorkspaceMember,
  type WorkspaceSummary,
} from "@/lib/workspace-api";

export default function IssuePage() {
  const { workspaceSlug, projectSlug, issueId } = useParams<{
    workspaceSlug: string;
    projectSlug: string;
    issueId: string;
  }>();
  const { protectedRequest, user } = useAuth();
  const baseURL = process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";
  const workspaceApi = useMemo(
    () => new WorkspaceApi(baseURL, protectedRequest),
    [baseURL, protectedRequest],
  );
  const projectApi = useMemo(
    () => new ProjectApi(baseURL, protectedRequest),
    [baseURL, protectedRequest],
  );
  const issueApi = useMemo(
    () => new IssueApi(baseURL, protectedRequest),
    [baseURL, protectedRequest],
  );
  const commentApi = useMemo(
    () => new CommentApi(baseURL, protectedRequest),
    [baseURL, protectedRequest],
  );
  const [workspace, setWorkspace] = useState<WorkspaceSummary | null>(null);
  const [issue, setIssue] = useState<Issue | null>(null);
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [comments, setComments] = useState<IssueComment[]>([]);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<IssueStatus>("TODO");
  const [priority, setPriority] = useState<IssuePriority>("MEDIUM");
  const [assigneeId, setAssigneeId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [commentBody, setCommentBody] = useState("");
  const [commentPending, setCommentPending] = useState(false);
  const [editingCommentId, setEditingCommentId] = useState<string | null>(null);
  const [editingCommentBody, setEditingCommentBody] = useState("");

  useEffect(() => {
    workspaceApi
      .list()
      .then(async (items) => {
        const selectedWorkspace = items.find(
          (item) => item.slug === workspaceSlug,
        );
        if (!selectedWorkspace) throw new Error("Workspace not found");
        const projects = await projectApi.list(selectedWorkspace.id);
        const selectedProject = projects.find(
          (item) => item.slug === projectSlug,
        );
        if (!selectedProject) throw new Error("Project not found");
        const [loadedIssue, workspaceMembers, loadedComments] =
          await Promise.all([
            issueApi.get(selectedWorkspace.id, selectedProject.slug, issueId),
            workspaceApi.members(selectedWorkspace.id),
            commentApi.list(
              selectedWorkspace.id,
              selectedProject.slug,
              issueId,
            ),
          ]);
        setWorkspace(selectedWorkspace);
        setIssue(loadedIssue);
        setMembers(workspaceMembers);
        setComments(loadedComments);
        setTitle(loadedIssue.title);
        setDescription(loadedIssue.description);
        setStatus(loadedIssue.status);
        setPriority(loadedIssue.priority);
        setAssigneeId(loadedIssue.assigneeId ?? "");
      })
      .catch((caught) =>
        setError(
          caught instanceof Error ? caught.message : "Unable to load issue.",
        ),
      );
  }, [
    commentApi,
    issueApi,
    issueId,
    projectApi,
    projectSlug,
    workspaceApi,
    workspaceSlug,
  ]);

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace || !issue) return;
    setPending(true);
    setError(null);
    try {
      const updated = await issueApi.update(
        workspace.id,
        projectSlug,
        issue.id,
        {
          title,
          description,
          status,
          priority,
          assigneeId: assigneeId || null,
        },
      );
      setIssue(updated);
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to save issue.",
      );
    } finally {
      setPending(false);
    }
  }

  async function addComment(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace || !issue || !commentBody.trim()) return;
    setCommentPending(true);
    setError(null);
    try {
      const comment = await commentApi.create(
        workspace.id,
        projectSlug,
        issue.id,
        {
          body: commentBody,
        },
      );
      setComments((current) => [...current, comment]);
      setCommentBody("");
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to add comment.",
      );
    } finally {
      setCommentPending(false);
    }
  }

  async function saveComment(commentId: string) {
    if (!workspace || !issue || !editingCommentBody.trim()) return;
    setCommentPending(true);
    setError(null);
    try {
      const updated = await commentApi.update(
        workspace.id,
        projectSlug,
        issue.id,
        commentId,
        {
          body: editingCommentBody,
        },
      );
      setComments((current) =>
        current.map((comment) =>
          comment.id === commentId ? updated : comment,
        ),
      );
      setEditingCommentId(null);
      setEditingCommentBody("");
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to save comment.",
      );
    } finally {
      setCommentPending(false);
    }
  }

  async function deleteComment(commentId: string) {
    if (!workspace || !issue || !window.confirm("Delete this comment?")) return;
    setCommentPending(true);
    setError(null);
    try {
      await commentApi.delete(workspace.id, projectSlug, issue.id, commentId);
      setComments((current) =>
        current.filter((comment) => comment.id !== commentId),
      );
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to delete comment.",
      );
    } finally {
      setCommentPending(false);
    }
  }

  if (error)
    return (
      <main className="mx-auto max-w-3xl px-6 py-12">
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      </main>
    );
  if (!workspace || !issue)
    return (
      <main className="grid min-h-dvh place-items-center text-sm text-muted-foreground">
        Loading issue…
      </main>
    );
  const canManage = workspace.role === "OWNER" || workspace.role === "ADMIN";
  const canEdit =
    canManage || (workspace.role === "MEMBER" && issue.assigneeId === user?.id);
  const assignableMembers = canManage
    ? members
    : members.filter((member) => member.userId === user?.id);

  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <Link
        href={`/${workspace.slug}/${projectSlug}`}
        className="text-sm underline"
      >
        Back to project
      </Link>
      <h1 className="mt-6 text-2xl font-semibold">{issue.title}</h1>
      {canEdit ? (
        <form onSubmit={save} className="mt-6 grid gap-3 rounded-lg border p-4">
          <label className="grid gap-1 text-sm font-medium">
            Title
            <input
              required
              maxLength={200}
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
            />
          </label>
          <label className="grid gap-1 text-sm font-medium">
            Description
            <textarea
              maxLength={10000}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm font-normal"
            />
          </label>
          <div className="flex flex-wrap gap-3">
            <label className="grid gap-1 text-sm font-medium">
              Status
              <select
                value={status}
                onChange={(event) =>
                  setStatus(event.target.value as IssueStatus)
                }
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              >
                <option value="TODO">To do</option>
                <option value="IN_PROGRESS">In progress</option>
                <option value="DONE">Done</option>
              </select>
            </label>
            <label className="grid gap-1 text-sm font-medium">
              Priority
              <select
                value={priority}
                onChange={(event) =>
                  setPriority(event.target.value as IssuePriority)
                }
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              >
                <option value="LOW">Low</option>
                <option value="MEDIUM">Medium</option>
                <option value="HIGH">High</option>
              </select>
            </label>
            <label className="grid gap-1 text-sm font-medium">
              Assignee
              <select
                value={assigneeId}
                onChange={(event) => setAssigneeId(event.target.value)}
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              >
                {canManage && <option value="">Unassigned</option>}
                {assignableMembers.map((member) => (
                  <option key={member.userId} value={member.userId}>
                    {member.name || member.email}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <button
            disabled={pending || !title.trim()}
            className="w-fit rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {pending ? "Saving…" : "Save issue"}
          </button>
        </form>
      ) : (
        <section className="mt-4 grid gap-3 rounded-lg border p-4 text-sm">
          <p className="whitespace-pre-wrap text-muted-foreground">
            {issue.description || "No description."}
          </p>
          <dl className="grid grid-cols-2 gap-3 text-muted-foreground">
            <div>
              <dt className="font-medium text-foreground">Status</dt>
              <dd>{issue.status.replace("_", " ")}</dd>
            </div>
            <div>
              <dt className="font-medium text-foreground">Priority</dt>
              <dd>{issue.priority}</dd>
            </div>
            <div>
              <dt className="font-medium text-foreground">Assignee</dt>
              <dd>
                {members.find((member) => member.userId === issue.assigneeId)
                  ?.name || "Unassigned"}
              </dd>
            </div>
            <div>
              <dt className="font-medium text-foreground">Updated</dt>
              <dd>
                {issue.updatedAt
                  ? new Date(issue.updatedAt).toLocaleString()
                  : "Unknown"}
              </dd>
            </div>
          </dl>
        </section>
      )}
      <section
        className="mt-8 grid gap-4 border-t pt-6"
        aria-labelledby="comments-heading"
      >
        <div className="flex items-baseline justify-between gap-3">
          <h2 id="comments-heading" className="text-lg font-semibold">
            Comments
          </h2>
          <span className="text-sm text-muted-foreground">
            {comments.length}
          </span>
        </div>
        {comments.length ? (
          <ol className="grid gap-4">
            {comments.map((comment) => {
              const isAuthor = comment.authorId === user?.id;
              const isEditing = editingCommentId === comment.id;
              return (
                <li
                  key={comment.id}
                  className="grid gap-2 border-b pb-4 text-sm last:border-0"
                >
                  <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
                    <p className="font-medium">
                      {isAuthor ? "You" : comment.authorName}
                    </p>
                    <time
                      className="text-xs text-muted-foreground"
                      dateTime={comment.createdAt}
                    >
                      {new Date(comment.createdAt).toLocaleString()}
                    </time>
                  </div>
                  {isEditing ? (
                    <>
                      <textarea
                        aria-label="Edit comment"
                        maxLength={10000}
                        value={editingCommentBody}
                        onChange={(event) =>
                          setEditingCommentBody(event.target.value)
                        }
                        className="min-h-20 rounded-md border bg-background px-3 py-2"
                      />
                      <div className="flex gap-2">
                        <button
                          type="button"
                          disabled={
                            commentPending || !editingCommentBody.trim()
                          }
                          onClick={() => saveComment(comment.id)}
                          className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground disabled:opacity-50"
                        >
                          Save comment
                        </button>
                        <button
                          type="button"
                          disabled={commentPending}
                          onClick={() => setEditingCommentId(null)}
                          className="rounded-md border px-3 py-1.5 text-sm disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    </>
                  ) : (
                    <p className="whitespace-pre-wrap text-muted-foreground">
                      {comment.body}
                    </p>
                  )}
                  {isAuthor && !isEditing && (
                    <div className="flex gap-3">
                      <button
                        type="button"
                        disabled={commentPending}
                        onClick={() => {
                          setEditingCommentId(comment.id);
                          setEditingCommentBody(comment.body);
                        }}
                        className="text-sm underline disabled:opacity-50"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        disabled={commentPending}
                        onClick={() => deleteComment(comment.id)}
                        className="text-sm text-destructive underline disabled:opacity-50"
                      >
                        Delete
                      </button>
                    </div>
                  )}
                </li>
              );
            })}
          </ol>
        ) : (
          <p className="text-sm text-muted-foreground">No comments yet.</p>
        )}
        {workspace.role !== "VIEWER" && (
          <form onSubmit={addComment} className="grid gap-2">
            <label className="grid gap-1 text-sm font-medium">
              Add a comment
              <textarea
                required
                maxLength={10000}
                value={commentBody}
                onChange={(event) => setCommentBody(event.target.value)}
                className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm font-normal"
              />
            </label>
            <button
              disabled={commentPending || !commentBody.trim()}
              className="w-fit rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
            >
              {commentPending ? "Saving…" : "Add comment"}
            </button>
          </form>
        )}
      </section>
    </main>
  );
}
