"use client";

import { useSyncExternalStore } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { KeyRound, LogOut, Monitor } from "lucide-react";
import { apiRequest, ApiError } from "@/lib/api";
import type { AuthMessageResponse, SessionResponse } from "@/lib/types";
import { Notice, SaveState, SettingsCard, SettingsPageTitle } from "@/components/settings/settings-ui";

const noopSubscribe = () => () => {};

function describeBrowser() {
  const ua = navigator.userAgent;
  const browser = /Edg\//.test(ua) ? "Edge" : /Firefox\//.test(ua) ? "Firefox" : /Chrome\//.test(ua) ? "Chrome" : /Safari\//.test(ua) ? "Safari" : "Browser";
  const os = /Windows/.test(ua) ? "Windows" : /Mac OS X/.test(ua) ? "macOS" : /Android/.test(ua) ? "Android" : /iPhone|iPad/.test(ua) ? "iOS" : /Linux/.test(ua) ? "Linux" : "";
  return os ? `${browser} on ${os}` : browser;
}

export function SecuritySettings() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const sessionQuery = useQuery({ queryKey: ["session"], queryFn: () => apiRequest<SessionResponse>("/api/auth/session") });
  const session = sessionQuery.data;
  const user = session?.user;
  const isDemo = session?.mode === "demo";
  const browser = useSyncExternalStore(noopSubscribe, describeBrowser, () => "This browser");

  const sendReset = useMutation({
    mutationFn: (email: string) =>
      apiRequest<AuthMessageResponse>("/api/auth/forgot-password", { method: "POST", body: JSON.stringify({ email }) }),
  });

  const logout = useMutation({
    mutationFn: () => apiRequest<{ ok: boolean }>("/api/auth/logout", { method: "POST", body: JSON.stringify({}) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["session"] });
      router.replace("/login");
    },
  });

  return (
    <div className="grid gap-5">
      <SettingsPageTitle title="Password and sessions" description="How you sign in, and where you're signed in." />

      {isDemo ? <Notice tone="info">This is the shared demo account, so password changes are turned off.</Notice> : null}

      <SettingsCard
        id="password"
        title="Password"
        description={
          <>
            We&apos;ll email a reset link to <span className="lp-mono text-[var(--lp-text)]">{user?.email}</span>. It works once and expires after 30 minutes.
          </>
        }
        hint={
          <SaveState
            saved={sendReset.isSuccess ? sendReset.data.message || "Check your inbox for the reset link." : null}
            error={sendReset.isError ? (sendReset.error as ApiError).message : null}
            idle="Passwords are 8 to 72 characters."
          />
        }
        action={
          <button
            type="button"
            className="console-btn console-btn-secondary"
            disabled={!user?.email || isDemo || sendReset.isPending}
            onClick={() => user?.email && sendReset.mutate(user.email)}
          >
            <KeyRound className="h-3.5 w-3.5" /> {sendReset.isPending ? "Sending" : "Send reset email"}
          </button>
        }
      />

      <SettingsCard
        id="sessions"
        title="Sessions"
        description="Sessions last 30 days. Signing out ends the session on this device."
        hint={logout.isError ? <span className="text-[#ff8fb3]">{(logout.error as ApiError).message}</span> : null}
        action={
          <button type="button" className="console-btn console-btn-secondary" disabled={logout.isPending} onClick={() => logout.mutate()}>
            <LogOut className="h-3.5 w-3.5" /> {logout.isPending ? "Signing out" : "Sign out"}
          </button>
        }
      >
        <div className="flex items-center gap-3 rounded-lg border border-[var(--lp-border)] bg-white/[0.02] px-4 py-3">
          <span className="grid h-9 w-9 shrink-0 place-items-center rounded-md border border-[var(--lp-border-strong)] bg-white/[0.03]">
            <Monitor className="h-4 w-4 text-[var(--lp-muted)]" />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-[13px]">{browser}</p>
            <p className="text-[12px] text-[var(--lp-subtle)]">{isDemo ? "Demo session" : "Signed in"} · this device</p>
          </div>
          <span className="shrink-0 rounded-md border border-[rgba(74,222,128,0.25)] bg-[rgba(74,222,128,0.08)] px-2 py-0.5 text-[12px] text-[#86efac]">Current</span>
        </div>
      </SettingsCard>
    </div>
  );
}
