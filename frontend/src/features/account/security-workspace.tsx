"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AuthMessageResponse, RuntimeHealthResponse, SessionResponse } from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

function sessionModeLabel(session?: SessionResponse) {
  if (!session?.authenticated) {
    return "Locked";
  }
  return session.mode === "demo" ? "Demo Session" : "Authenticated";
}

export function SecurityWorkspace() {
  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
  });

  const runtimeHealthQuery = useQuery({
    queryKey: ["runtime-health"],
    queryFn: () => apiRequest<RuntimeHealthResponse>("/api/healthz"),
  });

  const resendVerification = useMutation({
    mutationFn: (email: string) =>
      apiRequest<AuthMessageResponse>("/api/auth/resend-verification", {
        method: "POST",
        body: JSON.stringify({ email }),
      }),
  });

  const sendResetEmail = useMutation({
    mutationFn: (email: string) =>
      apiRequest<AuthMessageResponse>("/api/auth/forgot-password", {
        method: "POST",
        body: JSON.stringify({ email }),
      }),
  });

  const session = sessionQuery.data;
  const user = session?.user;
  const isDemo = session?.mode === "demo";
  const verified = Boolean(user?.email_verified_at);

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Account Panel</p>
            <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Security & Access</h2>
            <p className="mt-1 max-w-3xl text-sm text-slate-400">
              Review verification status, reset credentials, and prepare the account layer for stronger SaaS security controls.
            </p>
          </div>
          <div className="grid gap-2 sm:grid-cols-3">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Session Mode</p>
              <p className="mt-2 text-lg font-semibold text-white">{sessionModeLabel(session)}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Email Status</p>
              <p className="mt-2 text-lg font-semibold text-white">{verified ? "Verified" : "Pending"}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Primary Identity</p>
              <p className="mt-2 truncate text-lg font-semibold text-white">{user?.email ?? "-"}</p>
            </div>
          </div>
        </div>
      </section>

      {isDemo ? (
        <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
          Demo mode is active. Recovery and verification actions are visible for review, but they are blocked for the shared contributor session.
        </div>
      ) : null}

      {resendVerification.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(resendVerification.error as ApiError).message || "Unable to resend verification code."}
        </div>
      ) : null}
      {resendVerification.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {resendVerification.data.message}
          {resendVerification.data.preview_code ? ` OTP: ${resendVerification.data.preview_code}` : ""}
        </div>
      ) : null}
      {sendResetEmail.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(sendResetEmail.error as ApiError).message || "Unable to send password reset email."}
        </div>
      ) : null}
      {sendResetEmail.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {sendResetEmail.data.message}
        </div>
      ) : null}

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)]">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Verification</p>
          <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Identity Trust</h3>
          <div className="mt-5 grid gap-3">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Verification State</p>
              <p className="mt-2 text-lg font-semibold text-white">{verified ? "Verified" : "Pending"}</p>
              <p className="mt-1.5 text-sm text-slate-400">
                {verified
                  ? `Verified on ${formatDateTime(user?.email_verified_at ?? "")}`
                  : "The account should complete email verification before being treated as fully trusted in a SaaS workspace."}
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button
                type="button"
                variant="secondary"
                className="h-11 rounded-2xl px-5"
                disabled={isDemo || verified || !user?.email || resendVerification.isPending}
                onClick={() => user?.email && resendVerification.mutate(user.email)}
              >
                {resendVerification.isPending ? "Sending..." : "Resend Verification"}
              </Button>
              <Button
                type="button"
                variant="secondary"
                className="h-11 rounded-2xl px-5"
                disabled={isDemo || !user?.email || sendResetEmail.isPending}
                onClick={() => user?.email && sendResetEmail.mutate(user.email)}
              >
                {sendResetEmail.isPending ? "Sending..." : "Send Reset Email"}
              </Button>
            </div>
          </div>
        </Card>

        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Roadmap</p>
          <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Next Security Layer</h3>
          <div className="mt-5 grid gap-3">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Planned Controls</p>
              <ul className="mt-2 space-y-2 text-sm text-slate-300">
                <li>Two-factor authentication</li>
                <li>Active session history</li>
                <li>Recent login activity</li>
                <li>Device and browser trust</li>
                <li>Per-user API keys</li>
              </ul>
            </div>
            <div className="dashboard-note-panel p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Platform Runtime</p>
              <p className="mt-2 text-sm text-slate-300">
                {runtimeHealthQuery.isSuccess
                  ? `Runtime health last checked at ${formatDateTime(runtimeHealthQuery.data.time)}.`
                  : "Runtime health can be surfaced here to help correlate account-access issues with service availability."}
              </p>
            </div>
          </div>
        </Card>
      </section>
    </div>
  );
}
