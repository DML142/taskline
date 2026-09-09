"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useAuth } from "@/features/auth/auth-context";
import {
  WorkspaceApi,
  type WorkspaceMember,
  type WorkspaceSummary,
} from "@/lib/workspace-api";
import { InviteApi } from "@/lib/invite-api";

export default function WorkspacePage() {
  const params = useParams<{ workspaceId: string }>();
  const { protectedRequest } = useAuth();
  const api = useMemo(
    () =>
      new WorkspaceApi(
        process.env.NEXT_PUBLIC_API_URL ?? "/api/v1",
        protectedRequest,
      ),
    [protectedRequest],
  );
  const inviteApi = useMemo(
    () =>
      new InviteApi(
        process.env.NEXT_PUBLIC_API_URL ?? "/api/v1",
        protectedRequest,
      ),
    [protectedRequest],
  );
  const [workspace, setWorkspace] = useState<WorkspaceSummary | null>(null);
  const [members, setMembers] = useState<WorkspaceMember[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"ADMIN" | "MEMBER" | "VIEWER">("MEMBER");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    Promise.all([api.get(params.workspaceId), api.members(params.workspaceId)])
      .then(([nextWorkspace, nextMembers]) => {
        setWorkspace(nextWorkspace);
        setMembers(nextMembers);
      })
      .catch((caught) =>
        setError(
          caught instanceof Error
            ? caught.message
            : "Unable to load workspace.",
        ),
      );
  }, [api, params.workspaceId]);

  async function reloadMembers() {
    setMembers(await api.members(params.workspaceId));
  }

  async function addMember(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      await inviteApi.create(params.workspaceId, { email, role });
      setEmail("");
      await reloadMembers();
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to send invitation.",
      );
    } finally {
      setPending(false);
    }
  }

  async function removeMember(userId: string) {
    setError(null);
    try {
      await api.removeMember(params.workspaceId, userId);
      await reloadMembers();
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to remove member.",
      );
    }
  }

  if (error)
    return (
      <main className="mx-auto max-w-3xl px-6 py-12">
        <Link href="/" className="text-sm underline">
          Back to workspaces
        </Link>
        <p role="alert" className="mt-6 text-sm text-destructive">
          {error}
        </p>
      </main>
    );
  if (!workspace || !members)
    return (
      <main className="grid min-h-dvh place-items-center text-sm text-muted-foreground">
        Loading workspace…
      </main>
    );
  const owner = workspace.role === "OWNER";
  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <Link href="/" className="text-sm underline">
        Back to workspaces
      </Link>
      <h1 className="mt-6 text-2xl font-semibold">{workspace.name}</h1>
      <p className="mt-1 text-sm text-muted-foreground">Members</p>
      {error && (
        <p role="alert" className="mt-4 text-sm text-destructive">
          {error}
        </p>
      )}
      {owner && (
        <form
          onSubmit={addMember}
          className="mt-6 flex flex-wrap gap-2 rounded-lg border p-4"
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
            disabled={pending}
            className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {pending ? "Adding…" : "Add member"}
          </button>
        </form>
      )}
      <ul className="mt-6 divide-y rounded-lg border">
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
            </span>
            <span className="flex items-center gap-3">
              <span className="text-xs text-muted-foreground">
                {member.role}
              </span>
              {owner && member.role !== "OWNER" && (
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
    </main>
  );
}
