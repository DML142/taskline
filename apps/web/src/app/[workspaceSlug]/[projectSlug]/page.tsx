"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { ProjectApi, type Project } from "@/lib/project-api";
import { WorkspaceApi, type WorkspaceSummary } from "@/lib/workspace-api";

export default function ProjectPage() {
  const { workspaceSlug, projectSlug } = useParams<{
    workspaceSlug: string;
    projectSlug: string;
  }>();
  const router = useRouter();
  const { protectedRequest } = useAuth();
  const baseURL = process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";
  const workspaceApi = useMemo(
    () => new WorkspaceApi(baseURL, protectedRequest),
    [baseURL, protectedRequest],
  );
  const projectApi = useMemo(
    () => new ProjectApi(baseURL, protectedRequest),
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

  useEffect(() => {
    workspaceApi
      .list()
      .then((items) => {
        const selected = items.find((item) => item.slug === workspaceSlug);
        if (!selected) throw new Error("Workspace not found");
        setWorkspace(selected);
        return projectApi.list(selected.id);
      })
      .then((items) => {
        const selected = items.find((item) => item.slug === projectSlug);
        if (!selected) throw new Error("Project not found");
        setProject(selected);
        setName(selected.name);
        setSlug(selected.slug);
        setDescription(selected.description);
        setArchived(selected.archived);
      })
      .catch((caught) =>
        setError(
          caught instanceof Error ? caught.message : "Unable to load project.",
        ),
      );
  }, [projectApi, projectSlug, workspaceApi, workspaceSlug]);

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
      <p className="mt-6 rounded-lg border border-dashed px-6 py-12 text-center text-sm text-muted-foreground">
        Issues will appear here in the next phase.
      </p>
    </main>
  );
}
