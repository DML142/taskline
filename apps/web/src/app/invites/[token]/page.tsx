"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/features/auth/auth-context";
import { InviteApi, type InvitePreview } from "@/lib/invite-api";

export default function InvitePage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const { protectedRequest, user } = useAuth();
  const api = useMemo(
    () =>
      new InviteApi(
        process.env.NEXT_PUBLIC_API_URL ?? "/api/v1",
        protectedRequest,
      ),
    [protectedRequest],
  );
  const [invite, setInvite] = useState<InvitePreview | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    api
      .preview(token)
      .then(setInvite)
      .catch((caught) =>
        setError(
          caught instanceof Error
            ? caught.message
            : "Invitation is unavailable.",
        ),
      );
  }, [api, token]);
  async function accept() {
    try {
      await api.accept(token);
      router.push("/app");
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Invitation is unavailable.",
      );
    }
  }
  if (error)
    return (
      <main className="mx-auto max-w-lg px-6 py-16">
        <p role="alert">{error}</p>
      </main>
    );
  if (!invite)
    return (
      <main className="grid min-h-dvh place-items-center">
        Loading invitation…
      </main>
    );
  if (!user)
    return (
      <main className="mx-auto max-w-lg px-6 py-16">
        <h1 className="text-2xl font-semibold">Join {invite.workspaceName}</h1>
        <p className="mt-3">
          Sign in or register with the invited email address to accept this
          invitation.
        </p>
        <div className="mt-6 flex gap-3">
          <Link
            className="underline"
            href={`/login?next=/invites/${encodeURIComponent(token)}`}
          >
            Sign in
          </Link>
          <Link
            className="underline"
            href={`/register?next=/invites/${encodeURIComponent(token)}`}
          >
            Register
          </Link>
        </div>
      </main>
    );
  return (
    <main className="mx-auto max-w-lg px-6 py-16">
      <h1 className="text-2xl font-semibold">Join {invite.workspaceName}</h1>
      <p className="mt-3">You will join as {invite.role}.</p>
      <button
        onClick={() => void accept()}
        className="mt-6 rounded-md bg-primary px-4 py-2 text-primary-foreground"
      >
        Accept invitation
      </button>
    </main>
  );
}
