"use client";

import { useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AdminSystemResponse, RuntimeHealthResponse } from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";

function dependencyTone(healthy: boolean, enabled: boolean) {
  if (!enabled) {
    return "dashboard-chip dashboard-chip-muted";
  }
  return healthy ? "dashboard-chip dashboard-chip-success" : "dashboard-chip dashboard-chip-danger";
}

export function AdminSystemWorkspace() {
  const systemQuery = useQuery({
    queryKey: ["admin-system"],
    queryFn: () => apiRequest<AdminSystemResponse>("/api/admin/system"),
  });

  const healthQuery = useQuery({
    queryKey: ["runtime-health"],
    queryFn: () => apiRequest<RuntimeHealthResponse>("/api/healthz"),
  });

  const system = systemQuery.data;
  const health = healthQuery.data;

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Platform Admin</p>
        <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Admin System</h2>
        <p className="mt-1 max-w-3xl text-sm text-slate-400">
          Runtime dependency posture, feature flags, and platform-wide operational counts for the current SaaS control plane.
        </p>
      </section>

      {systemQuery.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(systemQuery.error as ApiError).message || "Unable to load admin system context."}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Users</p>
          <p className="mt-2 text-3xl font-semibold text-white">{system?.overview.user_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Accounts across the platform</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspaces</p>
          <p className="mt-2 text-3xl font-semibold text-white">{system?.overview.workspace_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Provisioned SaaS tenants</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Active Sessions</p>
          <p className="mt-2 text-3xl font-semibold text-white">{system?.overview.session_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Authenticated session count</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Pending Invitations</p>
          <p className="mt-2 text-3xl font-semibold text-white">{system?.overview.pending_invitation_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Open join invitations</p>
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)]">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Runtime</p>
          <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Dependency Health</h3>
          <div className="mt-5 grid gap-3 sm:grid-cols-2">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Database</p>
              <div className="mt-2 flex items-center gap-2">
                <span className={dependencyTone(Boolean(health?.database.healthy), Boolean(health?.database.enabled))}>
                  {health?.database.enabled ? (health?.database.healthy ? "Healthy" : "Unhealthy") : "Disabled"}
                </span>
                <span className="text-sm text-slate-400">{system?.runtime.database_driver ?? health?.database.driver ?? "-"}</span>
              </div>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Redis</p>
              <div className="mt-2 flex items-center gap-2">
                <span className={dependencyTone(Boolean(health?.redis.healthy), Boolean(health?.redis.enabled))}>
                  {health?.redis.enabled ? (health?.redis.healthy ? "Healthy" : "Unhealthy") : "Disabled"}
                </span>
              </div>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Object Storage</p>
              <div className="mt-2 flex items-center gap-2">
                <span className={dependencyTone(Boolean(health?.object_storage.healthy), Boolean(health?.object_storage.enabled))}>
                  {health?.object_storage.enabled ? (health?.object_storage.healthy ? "Healthy" : "Unhealthy") : "Disabled"}
                </span>
              </div>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Email Delivery</p>
              <div className="mt-2 flex items-center gap-2">
                <span className={dependencyTone(system?.runtime.email_delivery_enabled ?? false, true)}>
                  {system?.runtime.email_delivery_enabled ? "Enabled" : "Disabled"}
                </span>
              </div>
            </div>
          </div>
          <p className="mt-4 text-sm text-slate-400">
            {health?.time ? `Latest runtime probe: ${formatDateTime(health.time)}` : "Waiting for runtime health payload."}
          </p>
        </Card>

        <div className="grid gap-6">
          <Card className="rounded-[1.5rem]">
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Feature Flags</p>
            <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Platform Controls</h3>
            <div className="mt-5 grid gap-3">
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Demo Access</p>
                <p className="mt-2 text-lg font-semibold text-white">{system?.features.demo_enabled ? "Enabled" : "Disabled"}</p>
              </div>
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Session TTL</p>
                <p className="mt-2 text-lg font-semibold text-white">{system?.features.session_ttl_hours ?? "-"} hours</p>
              </div>
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Configured Admins</p>
                <p className="mt-2 text-lg font-semibold text-white">{system?.features.configured_admins ?? "-"}</p>
              </div>
              <div className="dashboard-note-panel p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Public Base URL</p>
                <p className="mt-2 break-all text-sm text-slate-300">{system?.features.public_base_url || "Not configured"}</p>
              </div>
            </div>
          </Card>
        </div>
      </section>
    </div>
  );
}
