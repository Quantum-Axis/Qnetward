"use client";

import { useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AdminOverviewResponse, AdminWorkspacesResponse } from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";

function billingTone(status: string) {
  switch (status) {
    case "active":
      return "dashboard-chip dashboard-chip-success";
    case "trial":
      return "dashboard-chip dashboard-chip-warning";
    case "past_due":
    case "suspended":
      return "dashboard-chip dashboard-chip-danger";
    default:
      return "dashboard-chip dashboard-chip-muted";
  }
}

export function AdminWorkspacesWorkspace() {
  const workspacesQuery = useQuery({
    queryKey: ["admin-workspaces"],
    queryFn: () => apiRequest<AdminWorkspacesResponse>("/api/admin/workspaces?limit=100"),
  });

  const overviewQuery = useQuery({
    queryKey: ["admin-overview"],
    queryFn: () => apiRequest<AdminOverviewResponse>("/api/admin/overview"),
  });

  const workspaces = workspacesQuery.data?.workspaces ?? [];
  const stats = {
    activeSessions: workspaces.reduce((total, workspace) => total + workspace.active_session_count, 0),
    pendingInvites: workspaces.reduce((total, workspace) => total + workspace.pending_invitation_count, 0),
    trialCount: workspaces.filter((workspace) => workspace.billing_status === "trial").length,
  };

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Platform Admin</p>
        <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Admin Workspaces</h2>
        <p className="mt-1 max-w-3xl text-sm text-slate-400">
          Tenant oversight across workspace plan posture, owner identity, pending invitations, and active session footprint.
        </p>
      </section>

      {workspacesQuery.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(workspacesQuery.error as ApiError).message || "Unable to load admin workspaces."}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspaces</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overviewQuery.data?.workspace_count ?? workspaces.length}</p>
          <p className="mt-2 text-sm text-slate-400">Provisioned tenants</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Pending Invites</p>
          <p className="mt-2 text-3xl font-semibold text-white">{stats.pendingInvites}</p>
          <p className="mt-2 text-sm text-slate-400">Invitation backlog across tenants</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Trial Workspaces</p>
          <p className="mt-2 text-3xl font-semibold text-white">{stats.trialCount}</p>
          <p className="mt-2 text-sm text-slate-400">Workspaces still on trial billing</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Active Sessions</p>
          <p className="mt-2 text-3xl font-semibold text-white">{stats.activeSessions}</p>
          <p className="mt-2 text-sm text-slate-400">Sessions scoped to active workspaces</p>
        </Card>
      </section>

      <div className="dashboard-table dashboard-table-shell rounded-[1.5rem]">
        <div className="flex items-center justify-between gap-3 border-b border-white/8 px-4 py-4 md:px-5">
          <div>
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Tenant Ledger</p>
            <h3 className="mt-1 text-lg font-semibold text-white">Workspace Inventory</h3>
          </div>
          <span className="dashboard-subtle-chip">{workspaces.length} loaded</span>
        </div>
        <div className="overflow-x-auto">
          <table className="data-table">
            <thead>
              <tr>
                <th>Workspace</th>
                <th>Owner</th>
                <th>Plan</th>
                <th>Billing</th>
                <th>Members</th>
                <th>Pending Invites</th>
                <th>Active Sessions</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {workspaces.length === 0 ? (
                <tr>
                  <td colSpan={8} className="text-sm text-slate-400">
                    No workspaces found yet.
                  </td>
                </tr>
              ) : (
                workspaces.map((workspace) => (
                  <tr key={workspace.id}>
                    <td data-label="Workspace">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-white">{workspace.name}</p>
                        <p className="truncate text-xs text-slate-400">ID {workspace.id}</p>
                      </div>
                    </td>
                    <td data-label="Owner">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-white">{workspace.owner_name}</p>
                        <p className="truncate text-xs text-slate-400">{workspace.owner_email}</p>
                      </div>
                    </td>
                    <td data-label="Plan">{workspace.plan}</td>
                    <td data-label="Billing">
                      <span className={billingTone(workspace.billing_status)}>{workspace.billing_status}</span>
                    </td>
                    <td data-label="Members">{workspace.member_count}</td>
                    <td data-label="Pending Invites">{workspace.pending_invitation_count}</td>
                    <td data-label="Active Sessions">{workspace.active_session_count}</td>
                    <td data-label="Created">{formatDateTime(workspace.created_at)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
