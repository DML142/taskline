"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { ProjectApi, type Project } from "@/lib/project-api";
import {
  IssueApi,
  type Issue,
  type IssueInput,
  type IssuePriority,
  type IssueStatus,
} from "@/lib/issue-api";
import {
  WorkspaceApi,
  type WorkspaceMember,
  type WorkspaceSummary,
} from "@/lib/workspace-api";

const boardColumns: ReadonlyArray<{ status: IssueStatus; title: string }> = [
  { status: "TODO", title: "To do" },
  { status: "IN_PROGRESS", title: "In progress" },
  { status: "DONE", title: "Done" },
];

type IssueFilters = {
  priority: "" | IssuePriority;
  assigneeId: string;
};

function matchesIssueFilters(issue: Issue, filters: IssueFilters) {
  return (
    (!filters.priority || issue.priority === filters.priority) &&
    (!filters.assigneeId || issue.assigneeId === filters.assigneeId)
  );
}

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
  const [visibleStatuses, setVisibleStatuses] = useState<IssueStatus[]>(
    boardColumns.map((column) => column.status),
  );
  const [priorityFilter, setPriorityFilter] = useState<"" | IssuePriority>("");
  const [assigneeFilter, setAssigneeFilter] = useState("");
  const [issueTitle, setIssueTitle] = useState("");
  const [issueDescription, setIssueDescription] = useState("");
  const [issuePriority, setIssuePriority] = useState<IssuePriority>("MEDIUM");
  const [issueAssignee, setIssueAssignee] = useState("");
  const [draggedIssueId, setDraggedIssueId] = useState<string | null>(null);
  const [activeDropStatus, setActiveDropStatus] = useState<IssueStatus | null>(
    null,
  );
  const [inFlightIssueIds, setInFlightIssueIds] = useState<Set<string>>(
    () => new Set(),
  );
  const listRequestGeneration = useRef(0);
  const filtersRef = useRef<IssueFilters>({
    priority: priorityFilter,
    assigneeId: assigneeFilter,
  });

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
    const requestGeneration = ++listRequestGeneration.current;
    issueApi
      .list(workspace.id, project.slug, {
        assigneeId: assigneeFilter || undefined,
        priority: priorityFilter || undefined,
      })
      .then((items) => {
        if (listRequestGeneration.current === requestGeneration) {
          setIssues(items);
        }
      })
      .catch((caught) => {
        if (listRequestGeneration.current === requestGeneration) {
          setError(
            caught instanceof Error ? caught.message : "Unable to load issues.",
          );
        }
      });
  }, [assigneeFilter, issueApi, priorityFilter, project, workspace]);

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
    listRequestGeneration.current += 1;
    setPending(true);
    setError(null);
    try {
      const created = await issueApi.create(workspace.id, project.slug, {
        title: issueTitle,
        description: issueDescription,
        status: "TODO",
        priority: issuePriority,
        assigneeId: issueAssignee || null,
      });
      setIssues((current) =>
        matchesCurrentFilters(created)
          ? [created, ...current.filter((issue) => issue.id !== created.id)]
          : current,
      );
      setIssueTitle("");
      setIssueDescription("");
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
  if (error && (!workspace || !project))
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
  const workspaceRole = workspace.role;
  const canManage = workspaceRole === "OWNER" || workspaceRole === "ADMIN";
  const canCreateIssue = canManage || workspaceRole === "MEMBER";
  const boardGridClass =
    {
      1: "md:grid-cols-1",
      2: "md:grid-cols-2",
      3: "md:grid-cols-3",
    }[visibleStatuses.length] ?? "md:grid-cols-1";
  const assignableMembers = canManage
    ? members
    : members.filter((member) => member.userId === user?.id);

  function canMoveIssue(issue: Issue) {
    return (
      !inFlightIssueIds.has(issue.id) &&
      (canManage ||
        (workspaceRole === "MEMBER" && issue.assigneeId === user?.id))
    );
  }

  function matchesCurrentFilters(issue: Issue) {
    return matchesIssueFilters(issue, filtersRef.current);
  }

  function replaceFilteredIssue(current: Issue[], nextIssue: Issue) {
    const withoutIssue = current.filter((issue) => issue.id !== nextIssue.id);
    return matchesCurrentFilters(nextIssue)
      ? [nextIssue, ...withoutIssue]
      : withoutIssue;
  }

  function clearDragState() {
    setDraggedIssueId(null);
    setActiveDropStatus(null);
  }

  function handleDragStart(
    event: React.DragEvent<HTMLAnchorElement>,
    issue: Issue,
  ) {
    if (!canMoveIssue(issue)) return;
    setError(null);
    setDraggedIssueId(issue.id);
    event.dataTransfer.setData("text/plain", issue.id);
  }

  function handleDragOver(
    event: React.DragEvent<HTMLElement>,
    status: IssueStatus,
  ) {
    const draggedIssue = issues.find((issue) => issue.id === draggedIssueId);
    if (!draggedIssue || !canMoveIssue(draggedIssue)) return;
    event.preventDefault();
    setActiveDropStatus(status);
  }

  function handleDragLeave(
    event: React.DragEvent<HTMLElement>,
    status: IssueStatus,
  ) {
    if (event.currentTarget !== event.target) return;
    if (event.currentTarget.contains(event.relatedTarget as Node | null))
      return;
    setActiveDropStatus((current) => (current === status ? null : current));
  }

  async function handleDrop(
    event: React.DragEvent<HTMLElement>,
    targetStatus: IssueStatus,
  ) {
    event.preventDefault();
    if (!workspace || !project) {
      clearDragState();
      return;
    }
    const issueId = draggedIssueId || event.dataTransfer.getData("text/plain");
    const issue = issues.find((item) => item.id === issueId);
    if (!issue || issue.status === targetStatus || !canMoveIssue(issue)) {
      clearDragState();
      return;
    }

    listRequestGeneration.current += 1;
    const optimisticIssue = { ...issue, status: targetStatus };
    const input: IssueInput = {
      title: issue.title,
      description: issue.description,
      priority: issue.priority,
      assigneeId: issue.assigneeId,
      status: targetStatus,
    };
    setError(null);
    setIssues((current) => replaceFilteredIssue(current, optimisticIssue));
    setInFlightIssueIds((current) => new Set(current).add(issue.id));
    clearDragState();
    try {
      const updated = await issueApi.update(
        workspace.id,
        project.slug,
        issue.id,
        input,
      );
      setIssues((current) => replaceFilteredIssue(current, updated));
    } catch (caught) {
      setIssues((current) => replaceFilteredIssue(current, issue));
      setError(
        caught instanceof Error ? caught.message : "Unable to update issue.",
      );
    } finally {
      setInFlightIssueIds((current) => {
        const next = new Set(current);
        next.delete(issue.id);
        return next;
      });
    }
  }

  return (
    <main className="mx-auto max-w-7xl px-6 py-12">
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
        {error && (
          <p role="alert" className="mb-4 text-sm text-destructive">
            {error}
          </p>
        )}
        <div className="flex flex-wrap items-end justify-between gap-3">
          <h2 className="text-lg font-semibold">Issues</h2>
          <div className="flex flex-wrap gap-3">
            <fieldset className="grid gap-1 text-sm font-medium">
              <legend>Visible columns</legend>
              <div className="flex flex-wrap gap-3">
                {boardColumns.map((column) => (
                  <label
                    key={column.status}
                    className="flex items-center gap-1 font-normal"
                  >
                    <input
                      type="checkbox"
                      aria-label={`Show ${column.title}`}
                      checked={visibleStatuses.includes(column.status)}
                      onChange={() =>
                        setVisibleStatuses((current) =>
                          current.includes(column.status)
                            ? current.filter(
                                (status) => status !== column.status,
                              )
                            : [...current, column.status],
                        )
                      }
                    />
                    <span aria-hidden="true">{column.title}</span>
                  </label>
                ))}
              </div>
            </fieldset>
            <label className="grid gap-1 text-sm font-medium">
              Filter by assignee
              <select
                value={assigneeFilter}
                onChange={(event) => {
                  const assigneeId = event.target.value;
                  filtersRef.current = { ...filtersRef.current, assigneeId };
                  setAssigneeFilter(assigneeId);
                }}
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
            <label className="grid gap-1 text-sm font-medium">
              Priority
              <select
                value={priorityFilter}
                onChange={(event) => {
                  const priority = event.target.value as "" | IssuePriority;
                  filtersRef.current = { ...filtersRef.current, priority };
                  setPriorityFilter(priority);
                }}
                className="rounded-md border bg-background px-3 py-2 text-sm font-normal"
              >
                <option value="">All priorities</option>
                <option value="LOW">Low</option>
                <option value="MEDIUM">Medium</option>
                <option value="HIGH">High</option>
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
                Issue priority
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
                Assign to
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
        <div
          data-testid="kanban-board"
          className={`mt-4 grid gap-4 ${boardGridClass}`}
        >
          {boardColumns
            .filter((column) => visibleStatuses.includes(column.status))
            .map((column) => {
              const columnIssues = issues.filter(
                (issue) => issue.status === column.status,
              );
              return (
                <section
                  key={column.status}
                  aria-label={column.title}
                  onDragOver={(event) => handleDragOver(event, column.status)}
                  onDragLeave={(event) => handleDragLeave(event, column.status)}
                  onDrop={(event) => handleDrop(event, column.status)}
                  className={`min-h-52 rounded-lg border p-3 ${
                    activeDropStatus === column.status
                      ? "border-primary bg-muted"
                      : "bg-muted/30"
                  }`}
                >
                  <h3 className="text-sm font-semibold">{column.title}</h3>
                  <div className="mt-3 grid gap-2">
                    {columnIssues.map((issue) => {
                      const member = members.find(
                        (member) => member.userId === issue.assigneeId,
                      );
                      const movable = canMoveIssue(issue);
                      return (
                        <Link
                          key={issue.id}
                          href={`/${workspace.slug}/${project.slug}/${issue.id}`}
                          draggable={movable}
                          onDragStart={(event) => handleDragStart(event, issue)}
                          onDragEnd={clearDragState}
                          className="grid gap-2 rounded-lg border bg-background px-4 py-3 hover:bg-muted"
                        >
                          <span className="font-medium">{issue.title}</span>
                          <span className="text-xs text-muted-foreground">
                            {issue.priority} ·{" "}
                            {member?.name || member?.email || "Unassigned"}
                          </span>
                        </Link>
                      );
                    })}
                    {columnIssues.length === 0 && (
                      <p className="rounded-md border border-dashed px-3 py-6 text-center text-xs text-muted-foreground">
                        No issues
                      </p>
                    )}
                  </div>
                </section>
              );
            })}
        </div>
      </section>
    </main>
  );
}
