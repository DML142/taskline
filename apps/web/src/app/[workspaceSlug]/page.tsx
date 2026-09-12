"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { ProjectApi, type Project } from "@/lib/project-api";
import {
  WorkspaceApi,
  type WorkspaceMember,
  type WorkspaceSummary,
} from "@/lib/workspace-api";
import { InviteApi, type WorkspaceInvite } from "@/lib/invite-api";

export default function WorkspaceProjectsPage() {
  const { workspaceSlug } = useParams<{ workspaceSlug: string }>();
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
  const inviteApi = useMemo(
    () => new InviteApi(baseURL, protectedRequest),
    [baseURL, protectedRequest],
  );
  const [workspace, setWorkspace] = useState<WorkspaceSummary | null>(null);
  const [projects, setProjects] = useState<Project[] | null>(null);
  const [members, setMembers] = useState<WorkspaceMember[] | null>(null);
  const [invites, setInvites] = useState<WorkspaceInvite[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"ADMIN" | "MEMBER" | "VIEWER">("MEMBER");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [memberPending, setMemberPending] = useState(false);

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
          inviteApi.list(selected.id),
        ]) as Promise<[Project[], WorkspaceMember[], WorkspaceInvite[]]>;
      })
      .then(([nextProjects, nextMembers, nextInvites]) => {
        setProjects(nextProjects);
        setMembers(nextMembers);
        setInvites(nextInvites);
      })
      .catch((caught) => setError(message(caught)));
  }, [inviteApi, projectApi, workspaceApi, workspaceSlug]);

  async function create(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace) return;
    setPending(true);
    setError(null);
    try {
      const project = await projectApi.create(workspace.id, {
        name,
        description,
      });
      setProjects((current) => [...(current ?? []), project]);
      setName("");
      setDescription("");
    } catch (caught) {
      setError(message(caught));
    } finally {
      setPending(false);
    }
  }

  async function reloadMembers() {
    if (!workspace) return;
    setMembers(await workspaceApi.members(workspace.id));
  }

  async function addMember(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace) return;
    setMemberPending(true);
    setError(null);
    try {
      await inviteApi.create(workspace.id, { email, role });
      setEmail("");
      setInvites(await inviteApi.list(workspace.id));
    } catch (caught) {
      setError(message(caught));
    } finally {
      setMemberPending(false);
    }
  }

  async function revokeInvite(inviteId: string) {
    if (!workspace) return;
    try {
      await inviteApi.revoke(workspace.id, inviteId);
      setInvites((current) =>
        current.filter((invite) => invite.id !== inviteId),
      );
    } catch (caught) {
      setError(message(caught));
    }
  }

  async function removeMember(userId: string) {
    if (!workspace) return;
    setError(null);
    try {
      await workspaceApi.removeMember(workspace.id, userId);
      await reloadMembers();
    } catch (caught) {
      setError(message(caught));
    }
  }

  if (error)
    return (
      <main className="mx-auto max-w-3xl px-6 py-12">
        <Link href="/app" className="text-sm underline">
          Back to workspaces
        </Link>
        <p role="alert" className="mt-6 text-sm text-destructive">
          {error}
        </p>
      </main>
    );
  if (!workspace || !projects || !members)
    return (
      <main className="grid min-h-dvh place-items-center text-sm text-muted-foreground">
        Loading projects…
      </main>
    );
  const canManage = workspace.role === "OWNER" || workspace.role === "ADMIN";
  const owner = workspace.role === "OWNER";
  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <Link href="/app" className="text-sm underline">
        Back to workspaces
      </Link>
      <h1 className="mt-6 text-2xl font-semibold">{workspace.name}</h1>
      <p className="mt-1 text-sm text-muted-foreground">Projects</p>
      {canManage && (
        <form
          onSubmit={create}
          className="mt-6 grid gap-3 rounded-lg border p-4"
        >
          <input
            required
            minLength={1}
            maxLength={120}
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="rounded-md border bg-background px-3 py-2 text-sm"
            placeholder="Project name"
            aria-label="Project name"
          />
          <textarea
            maxLength={2000}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            className="min-h-24 rounded-md border bg-background px-3 py-2 text-sm"
            placeholder="Description (optional)"
            aria-label="Project description"
          />
          <button
            disabled={pending || !name.trim()}
            className="w-fit rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {pending ? "Creating…" : "Create project"}
          </button>
        </form>
      )}
      <ul className="mt-6 grid gap-3">
        {projects.length === 0 ? (
          <li className="rounded-lg border border-dashed px-6 py-12 text-center text-sm text-muted-foreground">
            Create the first project in this workspace.
          </li>
        ) : (
          projects.map((project) => (
            <li key={project.id}>
              <Link
                href={`/${workspace.slug}/${project.slug}`}
                className="block rounded-lg border p-4 transition-colors hover:bg-muted"
              >
                <p className="font-medium">{project.name}</p>
                {project.description && (
                  <p className="mt-1 text-sm text-muted-foreground">
                    {project.description}
                  </p>
                )}
                {project.archived && (
                  <p className="mt-2 text-xs text-muted-foreground">Archived</p>
                )}
              </Link>
            </li>
          ))
        )}
      </ul>
      <section className="mt-10">
        <h2 className="text-lg font-semibold">Members</h2>
        {canManage && (
          <form
            onSubmit={addMember}
            className="mt-4 flex flex-wrap gap-2 rounded-lg border p-4"
          >
            <label className="sr-only" htmlFor="member-email">
              Member email
            </label>
            <input
              id="member-email"
              required
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              className="min-w-48 flex-1 rounded-md border bg-background px-3 py-2 text-sm"
              placeholder="person@example.com"
            />
            <label className="sr-only" htmlFor="member-role">
              Role
            </label>
            <select
              id="member-role"
              value={role}
              onChange={(event) => setRole(event.target.value as typeof role)}
              className="rounded-md border bg-background px-3 py-2 text-sm"
            >
              <option value="ADMIN">Admin</option>
              <option value="MEMBER">Member</option>
              <option value="VIEWER">Viewer</option>
            </select>
            <button
              disabled={memberPending}
              className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
            >
              {memberPending ? "Sending…" : "Send invitation"}
            </button>
          </form>
        )}
        <ul className="mt-4 divide-y rounded-lg border">
          {members.map((member) => (
            <li
              key={member.userId}
              className="flex items-center justify-between gap-3 p-4"
            >
              <span>
                <span className="block text-sm font-medium">{member.name}</span>
                <span className="block text-sm text-muted-foreground">
                  {member.email}
                </span>
                {canManage && member.addedByName && (
                  <span className="block text-xs text-muted-foreground">
                    Added by {member.addedByName}
                  </span>
                )}
              </span>
              <span className="flex items-center gap-3">
                <span className="text-xs text-muted-foreground">
                  {member.role}
                </span>
                {(owner ||
                  (workspace.role === "ADMIN" &&
                    (member.role !== "ADMIN" ||
                      member.addedByUserId === user?.id))) &&
                  member.role !== "OWNER" && (
                    <button
                      onClick={() => void removeMember(member.userId)}
                      className="text-xs text-destructive underline"
                    >
                      Remove
                    </button>
                  )}
              </span>
            </li>
          ))}
        </ul>
        {canManage && (
          <section className="mt-6">
            <h3 className="text-sm font-semibold">Active invitations</h3>
            <ul className="mt-2 divide-y rounded-lg border">
              {invites.length === 0 ? (
                <li className="p-4 text-sm text-muted-foreground">
                  No active invitations.
                </li>
              ) : (
                invites.map((invite) => (
                  <li
                    key={invite.id}
                    className="flex items-center justify-between gap-3 p-4 text-sm"
                  >
                    <span>
                      {invite.email} · {invite.role} · expires{" "}
                      {new Date(invite.expiresAt).toLocaleString()}
                    </span>
                    <button
                      onClick={() => void revokeInvite(invite.id)}
                      className="text-xs text-destructive underline"
                    >
                      Revoke
                    </button>
                  </li>
                ))
              )}
            </ul>
          </section>
        )}
      </section>
    </main>
  );
}

function message(caught: unknown) {
  return caught instanceof Error ? caught.message : "Unable to load projects.";
}
