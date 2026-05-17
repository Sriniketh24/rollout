"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, Flag, Loader2 } from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useDemoData } from "@/lib/use-demo-data";

export default function LoginPage() {
  const router = useRouter();
  const { signIn, signUp, isConfigured } = useAuth();
  const { startGuestWorkspace } = useDemoData();
  const [mode, setMode] = useState<"signin" | "signup">("signin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isStartingGuest, setIsStartingGuest] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setMessage(null);
    setIsSubmitting(true);

    try {
      if (mode === "signin") {
        await signIn(email, password);
        router.push("/");
        return;
      }

      const result = await signUp(email, password, displayName);
      if (result.needsEmailConfirmation) {
        setMessage("Check your email to confirm the account, then sign in.");
      } else {
        router.push("/");
      }
    } catch (authError) {
      setError(
        authError instanceof Error
          ? authError.message
          : "Authentication failed"
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleGuestWorkspace() {
    setError(null);
    setMessage(null);
    setIsStartingGuest(true);
    try {
      await startGuestWorkspace();
      router.push("/");
    } catch (guestError) {
      setError(
        guestError instanceof Error
          ? guestError.message
          : "Unable to start guest workspace"
      );
    } finally {
      setIsStartingGuest(false);
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-md flex-col justify-center">
      <Link
        href="/"
        className="mb-8 inline-flex items-center gap-2 text-sm text-zinc-400 transition-colors hover:text-zinc-200"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to public demo
      </Link>

      <div className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-6">
        <div className="mb-6 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-600">
            <Flag className="h-5 w-5 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-semibold text-zinc-100">
              {mode === "signin" ? "Sign in to Rollout" : "Create a workspace"}
            </h1>
            <p className="text-sm text-zinc-500">
              {mode === "signin"
                ? "Manage your own flags and experiments."
                : "Start with a private workspace and default environments."}
            </p>
          </div>
        </div>

        {!isConfigured && (
          <div className="mb-4 rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-300">
            Supabase is not configured for this deployment.
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          {mode === "signup" && (
            <div>
              <label className="mb-1.5 block text-sm font-medium text-zinc-300">
                Name
              </label>
              <input
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                required
                className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
                placeholder="Demo Admin"
              />
            </div>
          )}

          <div>
            <label className="mb-1.5 block text-sm font-medium text-zinc-300">
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label className="mb-1.5 block text-sm font-medium text-zinc-300">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
              minLength={6}
              className="w-full rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-blue-500/50 focus:outline-none focus:ring-1 focus:ring-blue-500/30"
              placeholder="At least 6 characters"
            />
          </div>

          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-300">
              {error}
            </div>
          )}
          {message && (
            <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-sm text-emerald-300">
              {message}
            </div>
          )}

          <button
            type="submit"
            disabled={isSubmitting || !isConfigured}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
            {mode === "signin" ? "Sign in" : "Create workspace"}
          </button>
        </form>

        <div className="my-5 flex items-center gap-3">
          <div className="h-px flex-1 bg-zinc-800" />
          <span className="text-xs text-zinc-500">or</span>
          <div className="h-px flex-1 bg-zinc-800" />
        </div>

        <button
          type="button"
          onClick={() => void handleGuestWorkspace()}
          disabled={isStartingGuest}
          className="flex w-full items-center justify-center gap-2 rounded-lg border border-zinc-700 px-4 py-2.5 text-sm font-medium text-zinc-200 transition-colors hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isStartingGuest && <Loader2 className="h-4 w-4 animate-spin" />}
          Continue with editable guest workspace
        </button>

        <div className="mt-5 border-t border-zinc-800 pt-5 text-center text-sm text-zinc-400">
          {mode === "signin" ? (
            <>
              Need an account?{" "}
              <button
                type="button"
                onClick={() => {
                  setMode("signup");
                  setError(null);
                  setMessage(null);
                }}
                className="font-medium text-blue-400 hover:text-blue-300"
              >
                Create one
              </button>
            </>
          ) : (
            <>
              Already have an account?{" "}
              <button
                type="button"
                onClick={() => {
                  setMode("signin");
                  setError(null);
                  setMessage(null);
                }}
                className="font-medium text-blue-400 hover:text-blue-300"
              >
                Sign in
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
