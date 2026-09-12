import Link from "next/link";
import { ArrowUpRight, Layers } from "lucide-react";

export default function AboutPage() {
  return (
    <main className="min-h-dvh bg-[#fbfcfe] text-[#1f2933]">
      <header className="border-b border-[#d9e0e8] bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3 sm:px-6 lg:px-8">
          <Link
            href="/"
            className="flex shrink-0 items-center gap-3 text-lg font-semibold tracking-tight focus-visible:outline-2 focus-visible:outline-offset-4"
          >
            <span className="grid size-8 place-items-center bg-[#287ab5] text-white">
              <Layers aria-hidden="true" className="size-4" />
            </span>
            Taskline
          </Link>
          <Link
            href="/register"
            className="rounded-sm bg-[#287ab5] px-3 py-2 text-sm font-semibold whitespace-nowrap text-white hover:bg-[#176da7] focus-visible:outline-2 focus-visible:outline-offset-4 sm:px-4"
          >
            Create account
          </Link>
        </div>
      </header>

      <section className="mx-auto max-w-3xl px-6 pb-20 pt-20 sm:pb-28 sm:pt-28 lg:px-8">
        <h1 className="text-4xl font-semibold tracking-[-0.04em] sm:text-5xl">
          About Taskline
        </h1>
        <div className="mt-8 space-y-5 text-lg leading-8 text-[#52606d]">
          <p>
            Taskline is a small issue-tracking application for teams that want a
            straightforward view of their projects and the work inside them.
          </p>
          <p>
            It is built with Go, PostgreSQL, and Next.js as an open-source
            portfolio project. The scope stays deliberately practical: shared
            workspaces, projects, issues, and comments.
          </p>
        </div>
        <a
          href="https://github.com/DML142/taskline"
          target="_blank"
          rel="noreferrer"
          className="mt-10 inline-flex items-center gap-2 rounded-sm border border-[#9cabb9] bg-white px-5 py-3 text-sm font-semibold text-[#243b53] hover:border-[#287ab5] hover:text-[#176da7] focus-visible:outline-2 focus-visible:outline-offset-4"
        >
          View the project on GitHub
          <ArrowUpRight aria-hidden="true" className="size-4" />
        </a>
      </section>
    </main>
  );
}
