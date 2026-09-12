"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";

export function AuthForm({ mode }: { mode: "login" | "register" }) {
  const { login, register } = useAuth();
  const router = useRouter();
  const requestedNext = useSearchParams().get("next");
  const next =
    requestedNext?.startsWith("/") && !requestedNext.startsWith("//")
      ? requestedNext
      : "/app";
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      const form = new FormData(event.currentTarget);
      const email = String(form.get("email") ?? "");
      const password = String(form.get("password") ?? "");
      if (mode === "register") {
        await register(String(form.get("name") ?? ""), email, password);
        router.replace(`/verify-email?email=${encodeURIComponent(email)}`);
      } else {
        await login(email, password);
        router.replace(next);
      }
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : "Unable to complete authentication.",
      );
    } finally {
      setPending(false);
    }
  }
  return (
    <main className="mx-auto flex min-h-dvh max-w-md items-center px-6">
      <form
        onSubmit={submit}
        className="w-full space-y-4 rounded-lg border p-6"
      >
        <h1 className="text-xl font-semibold">
          {mode === "login" ? "Sign in" : "Create account"}
        </h1>
        {mode === "register" && (
          <label className="block text-sm">
            Name
            <input
              required
              name="name"
              className="mt-1 w-full rounded border p-2"
            />
          </label>
        )}
        <label className="block text-sm">
          Email
          <input
            required
            type="email"
            name="email"
            className="mt-1 w-full rounded border p-2"
          />
        </label>
        <label className="block text-sm">
          Password
          <div className="mt-1 flex gap-2">
            <input
              required
              minLength={8}
              maxLength={32}
              type={showPassword ? "text" : "password"}
              name="password"
              className="w-full rounded border p-2"
            />
            <button
              type="button"
              className="rounded border px-3"
              onClick={() => setShowPassword((value) => !value)}
            >
              {showPassword ? "Hide" : "Show"}
            </button>
          </div>
          {mode === "register" && (
            <span className="mt-1 block text-xs text-muted-foreground">
              8–32 characters, including a letter and a digit; no four identical
              characters in a row.
            </span>
          )}
        </label>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <button
          disabled={pending}
          className="w-full rounded bg-primary px-3 py-2 text-primary-foreground"
        >
          {pending
            ? "Please wait…"
            : mode === "login"
              ? "Sign in"
              : "Create account"}
        </button>
        <p className="text-sm text-muted-foreground">
          {mode === "login" ? (
            <>
              New here?{" "}
              <Link href={`/register?next=${encodeURIComponent(next)}`}>
                Create an account
              </Link>
            </>
          ) : (
            <>
              Already have an account?{" "}
              <Link href={`/login?next=${encodeURIComponent(next)}`}>
                Sign in
              </Link>
            </>
          )}
        </p>
      </form>
    </main>
  );
}
