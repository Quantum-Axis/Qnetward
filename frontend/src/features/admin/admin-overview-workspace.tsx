"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AdminOverviewResponse, RuntimeHealthResponse } from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";

export function AdminOverviewWorkspace() {
  const overviewQuery = useQuery({
    queryKey: ["admin-overview"],
    queryFn: () => apiRequest<AdminOverviewResponse>("/api/admin/overview"),
  });

  const healthQuery = useQuery({
    queryKey: ["runtime-health"],
    queryFn: () => apiRequest<RuntimeHealthResponse>("/api/healthz"),
  });

  const overview = overviewQuery.data;

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Platform Admin</p>
        <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Admin Overview</h2>
        <p className="mt-1 max-w-3xl text-sm text-slate-400">
          Platform-level visibility across users, workspaces, invitations, and active sessions.
        </p>
      </section>

      {overviewQuery.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(overviewQuery.error as ApiError).message || "Unable to load admin overview."}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Users</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overview?.user_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">{overview?.verified_user_count ?? "-"} verified accounts</p>
          <Link href="/admin/users" className="dashboard-text-link mt-4 inline-flex text-sm">
            Open user directory
          </Link>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspaces</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overview?.workspace_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Total provisioned SaaS tenants</p>
          <Link href="/admin/workspaces" className="dashboard-text-link mt-4 inline-flex text-sm">
            Open workspace inventory
          </Link>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Sessions</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overview?.session_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Active authenticated sessions</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Invitations</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overview?.invitation_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">{overview?.pending_invitation_count ?? "-"} pending invitations</p>
        </Card>
        <Card className="rounded-[1.5rem] md:col-span-2">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Platform Runtime</p>
          <p className="mt-2 text-lg font-semibold text-white">
            {healthQuery.data?.ok ? "Operational" : "Unknown"}
          </p>
          <p className="mt-2 text-sm text-slate-400">
            {healthQuery.data?.time ? `Last health signal: ${formatDateTime(healthQuery.data.time)}` : "Waiting for health payload."}
          </p>
          <Link href="/admin/system" className="dashboard-text-link mt-4 inline-flex text-sm">
            Open system controls
          </Link>
        </Card>
      </section>
    </div>
  );
}
