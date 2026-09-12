import Link from "next/link";
import { ArrowRight, Layers } from "lucide-react";

export default function Home() {
  return (
    <main className="min-h-dvh bg-[#f5f5f0] text-[#172033]">
      <header className="mx-auto flex max-w-6xl items-center justify-between px-4 py-6 sm:px-6 lg:px-8">
        <Link
          href="/"
          className="flex items-center gap-2 text-lg font-semibold tracking-tight"
        >
          <span className="grid size-8 place-items-center bg-[#172033] text-[#f5f5f0]">
            <Layers aria-hidden="true" className="size-4" />
          </span>
          Taskline
        </Link>
        <nav
          aria-label="Account"
          className="flex items-center gap-2 text-sm font-medium sm:gap-4"
        >
          <Link
            href="/login"
            className="rounded-sm px-1 py-1.5 whitespace-nowrap underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-4 sm:px-2"
          >
            Sign in
          </Link>
          <Link
            href="/register"
            className="rounded-sm bg-[#2455d6] px-3 py-2 whitespace-nowrap text-white hover:bg-[#1d46b3] focus-visible:outline-2 focus-visible:outline-offset-4 sm:px-4"
          >
            Create account
          </Link>
        </nav>
      </header>

      <section className="mx-auto grid max-w-6xl gap-14 px-6 pb-20 pt-14 lg:grid-cols-[1.05fr_.95fr] lg:items-center lg:px-8 lg:pb-28 lg:pt-24">
        <div className="max-w-xl">
          <p className="text-sm font-medium text-[#2455d6]">
            Project work, without the fog
          </p>
          <h1 className="mt-5 text-5xl leading-[0.98] font-semibold tracking-[-0.055em] text-balance sm:text-6xl">
            Make the next task clear.
          </h1>
          <p className="mt-6 max-w-lg text-lg leading-8 text-[#4b5568]">
            Taskline gives a small team one shared place for projects, issues,
            and the decisions that keep work moving.
          </p>
          <div className="mt-8 flex flex-wrap items-center gap-4">
            <Link
              href="/register"
              className="inline-flex items-center gap-2 rounded-sm bg-[#2455d6] px-5 py-3 text-sm font-semibold text-white hover:bg-[#1d46b3] focus-visible:outline-2 focus-visible:outline-offset-4"
            >
              Create an account
              <ArrowRight aria-hidden="true" className="size-4" />
            </Link>
            <Link
              href="/login"
              className="rounded-sm px-2 py-2 text-sm font-semibold underline underline-offset-4 hover:text-[#2455d6] focus-visible:outline-2 focus-visible:outline-offset-4"
            >
              Sign in
            </Link>
          </div>
        </div>

        <div
          aria-label="Example project board"
          className="border border-[#172033]/15 bg-[#fffefa] p-4 shadow-[10px_10px_0_#dbe3ff] sm:p-6"
        >
          <div className="flex items-center justify-between border-b border-[#172033]/12 pb-4">
            <div>
              <p className="text-xs font-medium text-[#4b5568]">
                Website refresh
              </p>
              <p className="mt-1 text-lg font-semibold tracking-tight">
                This week
              </p>
            </div>
            <span className="rounded-full bg-[#e9edff] px-3 py-1 text-xs font-semibold text-[#2455d6]">
              3 open
            </span>
          </div>
          <ol className="mt-4 space-y-3">
            <li className="border-l-4 border-[#2455d6] bg-[#f4f6ff] px-4 py-3">
              <p className="text-sm font-semibold">Review launch copy</p>
              <p className="mt-1 text-xs text-[#4b5568]">In progress · Maya</p>
            </li>
            <li className="border-l-4 border-[#d5dae5] px-4 py-3">
              <p className="text-sm font-semibold">
                Confirm mobile breakpoints
              </p>
              <p className="mt-1 text-xs text-[#4b5568]">To do · Engineering</p>
            </li>
            <li className="border-l-4 border-[#4c9b72] bg-[#f2faf5] px-4 py-3">
              <p className="text-sm font-semibold">Publish the design notes</p>
              <p className="mt-1 text-xs text-[#4b5568]">Done · Eli</p>
            </li>
          </ol>
        </div>
      </section>

      <section className="border-t border-[#172033]/12 bg-[#e7ebf6]">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-10 sm:grid-cols-3 lg:px-8">
          <div>
            <h2 className="text-sm font-semibold">A shared starting point</h2>
            <p className="mt-2 text-sm leading-6 text-[#4b5568]">
              Keep every workspace and project easy to find.
            </p>
          </div>
          <div>
            <h2 className="text-sm font-semibold">Issues with context</h2>
            <p className="mt-2 text-sm leading-6 text-[#4b5568]">
              Capture what needs attention, who owns it, and what changed.
            </p>
          </div>
          <div>
            <h2 className="text-sm font-semibold">A simple way forward</h2>
            <p className="mt-2 text-sm leading-6 text-[#4b5568]">
              Use a compact board to move work from next to done.
            </p>
          </div>
        </div>
      </section>
    </main>
  );
}
