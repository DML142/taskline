"use client";

import Link from "next/link";
import { FolderKanban, Layers, LayoutDashboard } from "lucide-react";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/features/auth/auth-context";

export default function Home() {
  const { ready, user, logout } = useAuth();
  const router = useRouter();
  useEffect(() => { if (ready && !user) router.replace("/login"); }, [ready, router, user]);
  if (!ready || !user) return <main className="grid min-h-dvh place-items-center text-sm text-muted-foreground">Checking your session…</main>;
  return (
    <div className="min-h-dvh md:grid md:grid-cols-[220px_1fr]">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-10 focus:rounded-md focus:bg-background focus:p-3 focus:ring-2"
      >
        Skip to content
      </a>
      <aside className="border-b bg-sidebar p-4 md:border-r md:border-b-0">
        <Link
          href="/"
          className="flex w-fit items-center gap-2 rounded-sm text-base font-semibold focus-visible:outline-2 focus-visible:outline-offset-4"
        >
          <Layers aria-hidden="true" className="size-5" />
          Taskline
        </Link>
        <nav aria-label="Main navigation" className="mt-6">
          <Link
            href="/"
            aria-current="page"
            className="flex items-center gap-2 rounded-md bg-sidebar-accent px-3 py-2 text-sm font-medium focus-visible:outline-2"
          >
            <LayoutDashboard aria-hidden="true" className="size-4" />
            Overview
          </Link>
        </nav>
        <div className="mt-8 hidden px-3 md:block">
          <p className="text-xs font-medium text-muted-foreground">
            Workspaces
          </p>
          <p className="mt-3 text-sm text-muted-foreground">
            No workspace selected
          </p>
        </div>
      </aside>
      <main id="main" tabIndex={-1} className="min-w-0">
        <header className="flex flex-wrap items-center justify-between gap-3 border-b px-6 py-4">
          <h1 className="text-sm font-medium">Overview</h1>
          <div className="flex items-center gap-3"><Badge variant="outline">{user.name}</Badge><button onClick={() => void logout()} className="text-sm underline">Sign out</button></div>
        </header>
        <div className="mx-auto max-w-5xl px-6 py-8 md:px-10">
          <h2 className="text-xl font-semibold tracking-tight">
            Your workspace
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            A shared place for projects and issues.
          </p>
          <section
            aria-labelledby="workspace-heading"
            className="mt-8 flex min-h-72 flex-col items-center justify-center rounded-lg border border-dashed px-6 py-12 text-center"
          >
            <FolderKanban
              aria-hidden="true"
              className="mb-4 size-7 text-muted-foreground"
            />
            <h3 id="workspace-heading" className="text-sm font-medium">
              Projects will live here
            </h3>
            <p className="mt-2 max-w-sm text-sm leading-6 text-muted-foreground">
              Workspace and project management are not available yet.
            </p>
          </section>
        </div>
      </main>
    </div>
  );
}
