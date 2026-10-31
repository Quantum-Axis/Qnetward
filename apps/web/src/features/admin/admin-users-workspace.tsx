"use client";

import { useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AdminOverviewResponse, AdminUsersResponse } from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";

function verificationTone(verifiedAt?: string | null) {
  return verifiedAt ? "dashboard-chip dashboard-chip-success" : "dashboard-chip dashboard-chip-muted";
}

function platformRoleTone(role: "admin" | "user") {
  return role === "admin" ? "dashboard-chip dashboard-chip-warning" : "dashboard-chip dashboard-chip-muted";
}

export function AdminUsersWorkspace() {
  const usersQuery = useQuery({
    queryKey: ["admin-users"],
    queryFn: () => apiRequest<AdminUsersResponse>("/api/admin/users?limit=100"),
  });

  const overviewQuery = useQuery({
    queryKey: ["admin-overview"],
    queryFn: () => apiRequest<AdminOverviewResponse>("/api/admin/overview"),
  });

  const users = usersQuery.data?.users ?? [];
  const stats = {
    platformAdmins: users.filter((user) => user.platform_role === "admin").length,
    activeSessions: users.reduce((total, user) => total + user.active_session_count, 0),
  };

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Platform Admin</p>
        <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Admin Users</h2>
        <p className="mt-1 max-w-3xl text-sm text-slate-400">
          Global user directory across the SaaS platform, including verification state, workspace participation, and active session posture.
        </p>
      </section>

      {usersQuery.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(usersQuery.error as ApiError).message || "Unable to load admin users."}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Users</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overviewQuery.data?.user_count ?? users.length}</p>
          <p className="mt-2 text-sm text-slate-400">Global user accounts</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Verified</p>
          <p className="mt-2 text-3xl font-semibold text-white">{overviewQuery.data?.verified_user_count ?? "-"}</p>
          <p className="mt-2 text-sm text-slate-400">Verified identities</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Platform Admins</p>
          <p className="mt-2 text-3xl font-semibold text-white">{stats.platformAdmins}</p>
          <p className="mt-2 text-sm text-slate-400">Accounts with admin RBAC</p>
        </Card>
        <Card className="rounded-[1.5rem]">
          <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Active Sessions</p>
          <p className="mt-2 text-3xl font-semibold text-white">{stats.activeSessions}</p>
          <p className="mt-2 text-sm text-slate-400">Authenticated session volume</p>
        </Card>
      </section>

      <div className="dashboard-table dashboard-table-shell rounded-[1.5rem]">
        <div className="flex items-center justify-between gap-3 border-b border-white/8 px-4 py-4 md:px-5">
          <div>
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Directory</p>
            <h3 className="mt-1 text-lg font-semibold text-white">Platform Users</h3>
          </div>
          <span className="dashboard-subtle-chip">{users.length} loaded</span>
        </div>
        <div className="overflow-x-auto">
          <table className="data-table">
            <thead>
              <tr>
                <th>User</th>
                <th>Platform Role</th>
                <th>Verification</th>
                <th>Workspaces</th>
                <th>Active Sessions</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-sm text-slate-400">
                    No users found yet.
                  </td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr key={user.id}>
                    <td data-label="User">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-white">{user.name}</p>
                        <p className="truncate text-xs text-slate-400">{user.email}</p>
                      </div>
                    </td>
                    <td data-label="Platform Role">
                      <span className={platformRoleTone(user.platform_role)}>{user.platform_role}</span>
                    </td>
                    <td data-label="Verification">
                      <div className="flex flex-col gap-1">
                        <span className={verificationTone(user.email_verified_at)}>
                          {user.email_verified_at ? "Verified" : "Pending"}
                        </span>
                        <span className="text-xs text-slate-500">
                          {user.email_verified_at ? formatDateTime(user.email_verified_at) : "Awaiting email confirmation"}
                        </span>
                      </div>
                    </td>
                    <td data-label="Workspaces">
                      <div className="flex flex-col gap-1">
                        <span className="text-sm font-medium text-white">{user.workspace_count}</span>
                        <span className="text-xs text-slate-500">{user.owned_workspace_count} owned</span>
                      </div>
                    </td>
                    <td data-label="Active Sessions">{user.active_session_count}</td>
                    <td data-label="Created">{formatDateTime(user.created_at)}</td>
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
