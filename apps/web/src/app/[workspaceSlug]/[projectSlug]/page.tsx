"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { ProjectApi, type Project } from "@/lib/project-api";
import {
  IssueApi,
  type Issue,
  type IssuePriority,
  type IssueStatus,
} from "@/lib/issue-api";
import {
  WorkspaceApi,
  type WorkspaceMember,
  type WorkspaceSummary,
} from "@/lib/workspace-api";

export default function ProjectPage() {
  const { workspaceSlug, projectSlug } = useParams<{
    workspaceSlug: string;
    projectSlug: string;
  }>();
  const router = useRouter();
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
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [archived, setArchived] = useState(false);
  const [pending, setPending] = useState(false);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [statusFilter, setStatusFilter] = useState<"" | IssueStatus>("");
  const [assigneeFilter, setAssigneeFilter] = useState("");
  const [issueTitle, setIssueTitle] = useState("");
  const [issueDescription, setIssueDescription] = useState("");
  const [issueStatus, setIssueStatus] = useState<IssueStatus>("TODO");
  const [issuePriority, setIssuePriority] = useState<IssuePriority>("MEDIUM");
  const [issueAssignee, setIssueAssignee] = useState("");

  useEffect(() => {
    workspaceApi
      .list()
      .then((items) => {
        const selected = items.find((item) => item.slug === workspaceSlug);
        if (!selected) throw new Error("Workspace not found");
        setWorkspace(selected);
        return Promise.all([
          projectApi.list(selected.id),
          workspaceApi.members(selected.id),
        ] as const);
      })
      .then(([items, workspaceMembers]) => {
        const selected = items.find((item) => item.slug === projectSlug);
        if (!selected) throw new Error("Project not found");
        setProject(selected);
        setName(selected.name);
        setSlug(selected.slug);
        setDescription(selected.description);
        setArchived(selected.archived);
        setMembers(workspaceMembers);
      })
      .catch((caught) =>
        setError(
          caught instanceof Error ? caught.message : "Unable to load project.",
        ),
      );
  }, [projectApi, projectSlug, workspaceApi, workspaceSlug]);

  useEffect(() => {
    if (!workspace || !project) return;
    issueApi
      .list(workspace.id, project.slug, {
        status: statusFilter || undefined,
        assigneeId: assigneeFilter || undefined,
      })
      .then(setIssues)
      .catch((caught) =>
        setError(
          caught instanceof Error ? caught.message : "Unable to load issues.",
        ),
      );
  }, [assigneeFilter, issueApi, project, statusFilter, workspace]);

  async function update(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace || !project) return;
    setPending(true);
    setError(null);
    try {
      const updated = await projectApi.update(workspace.id, project.slug, {
        name,
        slug,
        description,
        archived,
      });
      setProject(updated);
      if (updated.slug !== project.slug) {
        router.replace(`/${workspace.slug}/${updated.slug}`);
      }
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to update project.",
      );
    } finally {
      setPending(false);
    }
  }

  async function createIssue(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace || !project) return;
    setPending(true);
    setError(null);
    try {
      const created = await issueApi.create(workspace.id, project.slug, {
        title: issueTitle,
        description: issueDescription,
        status: issueStatus,
        priority: issuePriority,
        assigneeId: issueAssignee || null,
      });
      setIssues((current) => [created, ...current]);
      setIssueTitle("");
      setIssueDescription("");
      setIssueStatus("TODO");
      setIssuePriority("MEDIUM");
      setIssueAssignee("");
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to create issue.",
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
  if (!workspace || !project)
    return (
      <main className="grid min-h-dvh place-items-center text-sm text-muted-foreground">
        Loading project…
      </main>
    );
  const canManage = workspace.role === "OWNER" || workspace.role === "ADMIN";
  const canCreateIssue = canManage || workspace.role === "MEMBER";
  const assignableMembers = canManage
    ? members
    : members.filter((member) => member.userId === user?.id);
  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <Link href={`/${workspace.slug}`} className="text-sm underline">
        Back to {workspace.name}
      </Link>
      <h1 className="mt-6 text-2xl font-semibold">{project.name}</h1>
      {canManage ? (
        <form
          onSubmit={update}
          className="mt-6 grid gap-3 rounded-lg border p-4"
        >
          <label className="grid gap-1 text-sm font-medium">
            Name
            <input
              required
              minLength={1}
              maxLength={120}
              value={name}
              onChange={(event) => setName(event.target.value)}
              className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
            />
          </label>
          <label className="grid gap-1 text-sm font-medium">
            URL name
            <input
              required
              minLength={2}
              maxLength={80}
              pattern="[a-z0-9]+(-[a-z0-9]+)*"
              value={slug}
              onChange={(event) => setSlug(event.target.value)}
              className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
            />
          </label>
          <label className="grid gap-1 text-sm font-medium">
            Description
            <textarea
              maxLength={2000}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm font-normal"
            />
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={archived}
              onChange={(event) => setArchived(event.target.checked)}
            />
            Archive this project
          </label>
          <button
            disabled={pending || !name.trim()}
            className="w-fit rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {pending ? "Saving…" : "Save changes"}
          </button>
        </form>
      ) : project.description ? (
        <p className="mt-3 whitespace-pre-wrap text-sm text-muted-foreground">
          {project.description}
        </p>
      ) : null}
      <section className="mt-8">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <h2 className="text-lg font-semibold">Issues</h2>
          <div className="flex flex-wrap gap-3">
            <label className="grid gap-1 text-sm font-medium">
              Filter status
              <select
                value={statusFilter}
                onChange={(event) =>
                  setStatusFilter(event.target.value as "" | IssueStatus)
                }
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              >
                <option value="">All statuses</option>
                <option value="TODO">To do</option>
                <option value="IN_PROGRESS">In progress</option>
                <option value="DONE">Done</option>
              </select>
            </label>
            <label className="grid gap-1 text-sm font-medium">
              Assignee
              <select
                value={assigneeFilter}
                onChange={(event) => setAssigneeFilter(event.target.value)}
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              >
                <option value="">All assignees</option>
                {members.map((member) => (
                  <option key={member.userId} value={member.userId}>
                    {member.name || member.email}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
        {canCreateIssue && (
          <form
            onSubmit={createIssue}
            className="mt-4 grid gap-3 rounded-lg border p-4"
          >
            <label className="grid gap-1 text-sm font-medium">
              Title
              <input
                required
                maxLength={200}
                value={issueTitle}
                onChange={(event) => setIssueTitle(event.target.value)}
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              />
            </label>
            <label className="grid gap-1 text-sm font-medium">
              Description
              <textarea
                maxLength={10000}
                value={issueDescription}
                onChange={(event) => setIssueDescription(event.target.value)}
                className="min-h-20 rounded-md border bg-background px-3 py-2 text-sm font-normal"
              />
            </label>
            <div className="flex flex-wrap gap-3">
              <label className="grid gap-1 text-sm font-medium">
                Status
                <select
                  value={issueStatus}
                  onChange={(event) =>
                    setIssueStatus(event.target.value as IssueStatus)
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
                  value={issuePriority}
                  onChange={(event) =>
                    setIssuePriority(event.target.value as IssuePriority)
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
                  value={issueAssignee}
                  onChange={(event) => setIssueAssignee(event.target.value)}
                  className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
                >
                  <option value="">Unassigned</option>
                  {assignableMembers.map((member) => (
                    <option key={member.userId} value={member.userId}>
                      {member.name || member.email}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <button
              disabled={pending || !issueTitle.trim()}
              className="w-fit rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
            >
              {pending ? "Creating…" : "Create issue"}
            </button>
          </form>
        )}
        <div className="mt-4 grid gap-2">
          {issues.map((issue) => (
            <Link
              key={issue.id}
              href={`/${workspace.slug}/${project.slug}/${issue.id}`}
              className="flex items-center justify-between rounded-lg border px-4 py-3 hover:bg-muted"
            >
              <span className="font-medium">{issue.title}</span>
              <span className="text-xs text-muted-foreground">
                {issue.status.replace("_", " ")} · {issue.priority}
              </span>
            </Link>
          ))}
          {issues.length === 0 && (
            <p className="rounded-lg border border-dashed px-6 py-10 text-center text-sm text-muted-foreground">
              No issues match these filters.
            </p>
          )}
        </div>
      </section>
    </main>
  );
}
