"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
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
  const [workspace, setWorkspace] = useState<WorkspaceSummary | null>(null);
  const [issue, setIssue] = useState<Issue | null>(null);
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<IssueStatus>("TODO");
  const [priority, setPriority] = useState<IssuePriority>("MEDIUM");
  const [assigneeId, setAssigneeId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

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
        const [loadedIssue, workspaceMembers] = await Promise.all([
          issueApi.get(selectedWorkspace.id, selectedProject.slug, issueId),
          workspaceApi.members(selectedWorkspace.id),
        ]);
        setWorkspace(selectedWorkspace);
        setIssue(loadedIssue);
        setMembers(workspaceMembers);
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
  }, [issueApi, issueId, projectApi, projectSlug, workspaceApi, workspaceSlug]);

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
    </main>
  );
}
