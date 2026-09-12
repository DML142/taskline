"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";
import { AuthApi } from "@/lib/auth-api";

const api = new AuthApi(process.env.NEXT_PUBLIC_API_URL ?? "/api/v1");

export default function VerifyEmailPage() {
  return (
    <Suspense>
      <VerifyEmailContent />
    </Suspense>
  );
}

function VerifyEmailContent() {
  const params = useSearchParams();
  const token = params.get("token");
  const [email, setEmail] = useState(params.get("email") ?? "");
  const [message, setMessage] = useState(
    token
      ? "Verifying your email…"
      : "Check your inbox for a verification link.",
  );
  const [resending, setResending] = useState(false);
  const verificationStarted = useRef(false);

  useEffect(() => {
    if (!token || verificationStarted.current) return;
    verificationStarted.current = true;
    api
      .verifyEmail(token)
      .then(() => setMessage("Your email is verified. You can now sign in."))
      .catch(() =>
        setMessage(
          "This verification link is unavailable. Request a new one below.",
        ),
      );
  }, [token]);

  async function resend() {
    setResending(true);
    await api.resendVerification(email).catch(() => undefined);
    setMessage("If an account needs verification, a new email is on its way.");
    setResending(false);
  }

  return (
    <main className="mx-auto flex min-h-dvh max-w-md items-center px-6">
      <section className="w-full space-y-4 rounded-lg border p-6">
        <h1 className="text-xl font-semibold">Verify your email</h1>
        <p className="text-sm text-muted-foreground">{message}</p>
        <label className="block text-sm">
          Email
          <input
            className="mt-1 w-full rounded border p-2"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
          />
        </label>
        <button
          className="w-full rounded border px-3 py-2"
          disabled={resending || !email}
          onClick={resend}
        >
          {resending ? "Sending…" : "Send a new link"}
        </button>
      </section>
    </main>
  );
}
