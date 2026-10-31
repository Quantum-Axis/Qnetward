"use client";

import { useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AlertRule, RecentHistoryTarget, SessionResponse, WorkspaceTeamResponse } from "@/lib/types";
import { Card } from "@/components/ui/card";

export function BillingWorkspace() {
  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
  });

  const workspaceTeamQuery = useQuery({
    queryKey: ["workspace-team"],
    queryFn: () => apiRequest<WorkspaceTeamResponse>("/api/workspace/team"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const recentTargetsQuery = useQuery({
    queryKey: ["recent-history-targets", 12],
    queryFn: () => apiRequest<RecentHistoryTarget[]>("/api/history/recent-targets?limit=12"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const alertRulesQuery = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => apiRequest<AlertRule[]>("/api/alerts/rules"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const workspace = workspaceTeamQuery.data?.workspace;
  const members = workspaceTeamQuery.data?.members ?? [];
  const invitations = workspaceTeamQuery.data?.invitations ?? [];
  const targets = recentTargetsQuery.data ?? [];
  const alertRules = alertRulesQuery.data ?? [];
  const isDemo = sessionQuery.data?.mode === "demo";
  const enabledRules = alertRules.filter((rule) => rule.enabled).length;

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Workspace Panel</p>
            <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Billing & Usage</h2>
            <p className="mt-1 max-w-3xl text-sm text-slate-400">
              Track current plan posture, workspace usage signals, and billing readiness for the active SaaS tenant.
            </p>
          </div>
          <div className="grid gap-2 sm:grid-cols-3">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Plan</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspace?.plan ?? "starter"}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Billing State</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspace?.billing_status ?? "trial"}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspace Seats</p>
              <p className="mt-2 text-lg font-semibold text-white">{members.length}</p>
            </div>
          </div>
        </div>
      </section>

      {workspaceTeamQuery.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(workspaceTeamQuery.error as ApiError).message || "Unable to load workspace billing context."}
        </div>
      ) : null}

      {isDemo ? (
        <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
          Demo mode is active. Billing actions are visible for review, but purchase or plan mutation flows are intentionally disabled.
        </div>
      ) : null}

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)]">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Subscription</p>
          <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Current Plan Posture</h3>
          <div className="mt-5 grid gap-3 sm:grid-cols-2">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspace</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspace?.name ?? "Workspace loading"}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Status</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspace?.billing_status ?? "trial"}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Plan Tier</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspace?.plan ?? "starter"}</p>
              <p className="mt-1.5 text-sm text-slate-400">Stripe-ready plan mapping can be layered on this record later.</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Pending Seats</p>
              <p className="mt-2 text-lg font-semibold text-white">{invitations.filter((invite) => invite.status === "pending").length}</p>
              <p className="mt-1.5 text-sm text-slate-400">Invited users that may turn into billable seats once accepted.</p>
            </div>
          </div>
        </Card>

        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Usage</p>
          <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Workspace Usage Signals</h3>
          <div className="mt-5 grid gap-3">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Tracked Targets</p>
              <p className="mt-2 text-lg font-semibold text-white">{targets.length}</p>
              <p className="mt-1.5 text-sm text-slate-400">Recent telemetry targets visible from the workspace history feed.</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Configured Alerts</p>
              <p className="mt-2 text-lg font-semibold text-white">{enabledRules}</p>
              <p className="mt-1.5 text-sm text-slate-400">Enabled workspace alert rules currently counted against this tenant.</p>
            </div>
            <div className="dashboard-note-panel p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Next SaaS Billing Layer</p>
              <p className="mt-2 text-sm text-slate-300">
                Add Stripe-backed subscriptions, usage metering, invoice history, and seat-based limits on top of the current workspace plan model.
              </p>
            </div>
          </div>
        </Card>
      </section>
    </div>
  );
}
