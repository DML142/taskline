"use client";

import Link from "next/link";
import { Layers, LayoutDashboard } from "lucide-react";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/features/auth/auth-context";
import { WorkspaceOverview } from "@/features/workspaces/workspace-overview";

export default function AppPage() {
  const { ready, user, logout } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (ready && !user) router.replace("/login");
  }, [ready, router, user]);

  if (!ready || !user) {
    return (
      <main className="grid min-h-dvh place-items-center text-sm text-muted-foreground">
        Checking your session…
      </main>
    );
  }

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
          href="/app"
          className="flex w-fit items-center gap-2 rounded-sm text-base font-semibold focus-visible:outline-2 focus-visible:outline-offset-4"
        >
          <Layers aria-hidden="true" className="size-5" />
          Taskline
        </Link>
        <nav aria-label="Main navigation" className="mt-6">
          <Link
            href="/app"
            aria-current="page"
            className="flex items-center gap-2 rounded-md bg-sidebar-accent px-3 py-2 text-sm font-medium focus-visible:outline-2"
          >
            <LayoutDashboard aria-hidden="true" className="size-4" />
            Overview
          </Link>
        </nav>
      </aside>
      <main id="main" tabIndex={-1} className="min-w-0">
        <header className="flex flex-wrap items-center justify-between gap-3 border-b px-6 py-4">
          <h1 className="text-sm font-medium">Overview</h1>
          <div className="flex items-center gap-3">
            <Badge variant="outline">{user.name}</Badge>
            <button onClick={() => void logout()} className="text-sm underline">
              Sign out
            </button>
          </div>
        </header>
        <WorkspaceOverview />
      </main>
    </div>
  );
}
