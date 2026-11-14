"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { usePathname } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Shield } from "@/components/icons";
import { apiRequest } from "@/lib/api";
import type { WorkspaceContextResponse } from "@/lib/types";

/**
 * A workspace that requires two-factor sign-in keeps its pages from members
 * without it (the server refuses their requests too). Settings stay open, so
 * they can turn it on, or switch to another workspace.
 */
export function MFAGate({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const workspace = useQuery({
    queryKey: ["workspace-general"],
    queryFn: () => apiRequest<WorkspaceContextResponse>("/api/workspace"),
    staleTime: 60_000,
  });
  const required = workspace.data?.require_mfa === true;
  const mfa = useQuery({
    queryKey: ["account-mfa"],
    queryFn: () => apiRequest<{ enabled: boolean }>("/api/account/mfa"),
    enabled: required,
  });
  const open = pathname.startsWith("/settings") || pathname.startsWith("/workspace/team");
  if (!required || open || mfa.data?.enabled !== false) return <>{children}</>;

  return (
    <div className="mx-auto mt-16 flex max-w-md flex-col items-center text-center">
      <span className="grid h-12 w-12 place-items-center rounded-full border border-[var(--lp-border-strong)] bg-[#0a0a0a]">
        <Shield className="h-5 w-5 text-[var(--lp-text)]" />
      </span>
      <h1 className="mt-5 text-[22px] font-semibold tracking-[-0.02em]">Two-factor sign-in required</h1>
      <p className="mt-2 text-sm leading-relaxed text-[var(--lp-muted)]">
        {workspace.data?.workspace?.name ?? "This workspace"} asks every member to sign in with a second factor. Turn it on for your account to continue; it takes about a minute with an authenticator app.
      </p>
      <div className="mt-6 flex flex-wrap justify-center gap-2">
        <Link href="/settings/security" className="console-btn console-btn-primary h-9 px-4">
          Turn on two-factor
        </Link>
        <Link href="/settings/workspaces" className="console-btn console-btn-secondary h-9 px-4">
          Switch workspace
        </Link>
      </div>
    </div>
  );
}
