"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { WorkspaceApi, type WorkspaceSummary } from "@/lib/workspace-api";
import { useAuth } from "@/features/auth/auth-context";

export function WorkspaceOverview() {
  const { protectedRequest } = useAuth();
  const api = useMemo(
    () =>
      new WorkspaceApi(
        process.env.NEXT_PUBLIC_API_URL ?? "/api/v1",
        protectedRequest,
      ),
    [protectedRequest],
  );
  const [workspaces, setWorkspaces] = useState<WorkspaceSummary[] | null>(null);
  const [name, setName] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .list()
      .then(setWorkspaces)
      .catch((caught) => setError(message(caught)));
  }, [api]);

  async function create(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      const workspace = await api.create(name.trim());
      setWorkspaces((current) => [...(current ?? []), workspace]);
      setName("");
    } catch (caught) {
      setError(message(caught));
    } finally {
      setPending(false);
    }
  }

  return (
    <section className="mx-auto max-w-5xl px-6 py-8 md:px-10">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">Workspaces</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Shared places for your projects and issues.
          </p>
        </div>
        <form
          onSubmit={create}
          className="flex flex-wrap gap-2"
          aria-label="Create workspace"
        >
          <label className="sr-only" htmlFor="workspace-name">
            Workspace name
          </label>
          <input
            id="workspace-name"
            required
            minLength={1}
            maxLength={120}
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="rounded-md border bg-background px-3 py-2 text-sm"
            placeholder="Workspace name"
          />
          <button
            disabled={pending || !name.trim()}
            className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {pending ? "Creating…" : "Create workspace"}
          </button>
        </form>
      </div>
      {error ? (
        <p role="alert" className="mt-4 text-sm text-destructive">
          {error}
        </p>
      ) : workspaces === null ? (
        <p className="mt-8 text-sm text-muted-foreground">
          Loading workspaces…
        </p>
      ) : workspaces.length === 0 ? (
        <p className="mt-8 rounded-lg border border-dashed px-6 py-12 text-center text-sm text-muted-foreground">
          Create your first workspace to begin organizing work.
        </p>
      ) : (
        <ul className="mt-8 grid gap-3 sm:grid-cols-2">
          {workspaces.map((workspace) => (
            <li key={workspace.id}>
              <Link
                href={`/workspaces/${workspace.id}`}
                className="block rounded-lg border p-4 transition-colors hover:bg-muted"
              >
                <p className="font-medium">{workspace.name}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {workspace.role}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function message(caught: unknown) {
  return caught instanceof Error
    ? caught.message
    : "Unable to load workspaces.";
}
