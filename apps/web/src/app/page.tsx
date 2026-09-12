import Link from "next/link";
import { ArrowRight, Layers } from "lucide-react";

export default function Home() {
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
          <nav
            aria-label="Account"
            className="flex items-center gap-2 text-sm font-medium sm:gap-4"
          >
            <Link
              href="/login"
              className="rounded-sm px-1 py-1.5 whitespace-nowrap text-[#374151] hover:text-[#176da7] focus-visible:outline-2 focus-visible:outline-offset-4 sm:px-2"
            >
              Sign in
            </Link>
            <Link
              href="/register"
              className="rounded-sm bg-[#287ab5] px-3 py-2 whitespace-nowrap text-white hover:bg-[#176da7] focus-visible:outline-2 focus-visible:outline-offset-4 sm:px-4"
            >
              Create account
            </Link>
          </nav>
        </div>
      </header>

      <section className="mx-auto max-w-6xl px-6 pb-20 pt-24 sm:pb-28 sm:pt-32 lg:px-8">
        <div className="max-w-3xl border-l-4 border-[#287ab5] pl-6 sm:pl-8">
          <h1 className="text-5xl leading-[1.02] font-semibold tracking-[-0.045em] text-balance sm:text-6xl">
            Make the next task clear.
          </h1>
          <p className="mt-7 max-w-2xl text-lg leading-8 text-[#52606d]">
            Taskline is a focused home for the projects, issues, and decisions
            that help a small team move work forward.
          </p>
          <div className="mt-9 flex flex-wrap gap-3">
            <Link
              href="/register"
              className="inline-flex items-center gap-2 rounded-sm bg-[#287ab5] px-5 py-3 text-sm font-semibold text-white hover:bg-[#176da7] focus-visible:outline-2 focus-visible:outline-offset-4"
            >
              Create an account
              <ArrowRight aria-hidden="true" className="size-4" />
            </Link>
            <Link
              href="/about"
              className="rounded-sm border border-[#9cabb9] bg-white px-5 py-3 text-sm font-semibold text-[#243b53] hover:border-[#287ab5] hover:text-[#176da7] focus-visible:outline-2 focus-visible:outline-offset-4"
            >
              Learn more
            </Link>
          </div>
        </div>
      </section>
    </main>
  );
}
