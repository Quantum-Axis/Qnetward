"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import { AuthMessageResponse, SessionResponse, WorkspaceTeamResponse } from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

function buildInitials(name?: string | null, email?: string | null) {
  const source = (name || email || "NW").trim();
  const parts = source.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
  }
  return source.slice(0, 2).toUpperCase();
}

function workspaceRoleLabel(role: string) {
  switch (role) {
    case "owner":
      return "Workspace Owner";
    case "admin":
      return "Workspace Admin";
    case "viewer":
      return "Workspace Viewer";
    case "member":
      return "Workspace Member";
    default:
      return role;
  }
}

function invitationTone(status: string) {
  switch (status) {
    case "accepted":
      return "dashboard-chip dashboard-chip-success";
    case "expired":
      return "dashboard-chip dashboard-chip-danger";
    default:
      return "dashboard-chip dashboard-chip-muted";
  }
}

function canManageWorkspaceMember(actorRole: string, isDemo: boolean, isCurrent: boolean, targetRole: string) {
  if (isDemo || isCurrent) {
    return false;
  }
  if (actorRole === "owner") {
    return targetRole !== "owner";
  }
  if (actorRole === "admin") {
    return targetRole === "member" || targetRole === "viewer";
  }
  return false;
}

function memberRoleOptions(actorRole: string) {
  if (actorRole === "owner") {
    return ["admin", "member", "viewer"];
  }
  if (actorRole === "admin") {
    return ["member", "viewer"];
  }
  return [];
}

function invitationRoleOptions(actorRole: string) {
  if (actorRole === "owner") {
    return ["owner", "admin", "member", "viewer"];
  }
  if (actorRole === "admin") {
    return ["member", "viewer"];
  }
  return [];
}

function canManageInvitations(actorRole: string, isDemo: boolean) {
  return !isDemo && (actorRole === "owner" || actorRole === "admin");
}

export function TeamWorkspace() {
  const queryClient = useQueryClient();
  const [inviteValues, setInviteValues] = useState({ email: "", role: "member" });
  const [memberRoleDrafts, setMemberRoleDrafts] = useState<Record<number, string>>({});

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
  });

  const workspaceTeamQuery = useQuery({
    queryKey: ["workspace-team"],
    queryFn: () => apiRequest<WorkspaceTeamResponse>("/api/workspace/team"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const inviteMember = useMutation({
    mutationFn: (values: { email: string; role: string }) =>
      apiRequest<AuthMessageResponse>("/api/workspace/team", {
        method: "POST",
        body: JSON.stringify(values),
      }),
    onSuccess: async () => {
      setInviteValues({ email: "", role: "member" });
      await queryClient.invalidateQueries({ queryKey: ["workspace-team"] });
    },
  });

  const updateMemberRole = useMutation({
    mutationFn: (values: { userID: number; role: string }) =>
      apiRequest<AuthMessageResponse>("/api/workspace/team", {
        method: "PATCH",
        body: JSON.stringify({ user_id: values.userID, role: values.role }),
      }),
    onSuccess: async (_, variables) => {
      setMemberRoleDrafts((current) => {
        const next = { ...current };
        delete next[variables.userID];
        return next;
      });
      await queryClient.invalidateQueries({ queryKey: ["workspace-team"] });
    },
  });

  const removeMember = useMutation({
    mutationFn: (userID: number) =>
      apiRequest<AuthMessageResponse>("/api/workspace/team", {
        method: "DELETE",
        body: JSON.stringify({ user_id: userID }),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["workspace-team"] });
    },
  });

  const revokeInvitation = useMutation({
    mutationFn: (invitationID: number) =>
      apiRequest<AuthMessageResponse>("/api/workspace/team", {
        method: "DELETE",
        body: JSON.stringify({ invitation_id: invitationID }),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["workspace-team"] });
    },
  });

  const isDemo = sessionQuery.data?.mode === "demo";
  const workspace = workspaceTeamQuery.data?.workspace;
  const members = workspaceTeamQuery.data?.members ?? [];
  const invitations = workspaceTeamQuery.data?.invitations ?? [];
  const workspaceRole = workspace?.role ?? "member";
  const availableRoleOptions = memberRoleOptions(workspaceRole);
  const availableInvitationRoles = invitationRoleOptions(workspaceRole);
  const selectedInviteRole = availableInvitationRoles.includes(inviteValues.role) ? inviteValues.role : (availableInvitationRoles[0] ?? "member");
  const canManageWorkspaceInvitations = canManageInvitations(workspaceRole, isDemo);

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Workspace Panel</p>
            <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">Team Management</h2>
            <p className="mt-1 max-w-3xl text-sm text-slate-400">
              Manage workspace members, issue invitations, and control team-level access inside the active company workspace.
            </p>
          </div>
          <div className="grid gap-2 sm:grid-cols-3">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspace</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspace?.name ?? "Loading"}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Your Role</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspaceRoleLabel(workspaceRole)}</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Member Count</p>
              <p className="mt-2 text-lg font-semibold text-white">{members.length}</p>
            </div>
          </div>
        </div>
      </section>

      {isDemo ? (
        <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
          Demo mode is active. Team mutation actions are disabled for the shared contributor account.
        </div>
      ) : null}

      {inviteMember.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(inviteMember.error as ApiError).message || "Unable to send workspace invitation."}
        </div>
      ) : null}
      {inviteMember.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {inviteMember.data.message}
        </div>
      ) : null}
      {updateMemberRole.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(updateMemberRole.error as ApiError).message || "Unable to update workspace member role."}
        </div>
      ) : null}
      {updateMemberRole.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {updateMemberRole.data.message}
        </div>
      ) : null}
      {removeMember.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(removeMember.error as ApiError).message || "Unable to remove workspace member."}
        </div>
      ) : null}
      {removeMember.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {removeMember.data.message}
        </div>
      ) : null}
      {revokeInvitation.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(revokeInvitation.error as ApiError).message || "Unable to revoke workspace invitation."}
        </div>
      ) : null}
      {revokeInvitation.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {revokeInvitation.data.message}
        </div>
      ) : null}

      <section className="grid gap-6 2xl:grid-cols-[minmax(0,1.1fr)_minmax(360px,0.9fr)]">
        <Card className="rounded-[1.5rem]">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Roster</p>
              <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Active Team Members</h3>
            </div>
            {workspaceTeamQuery.isLoading ? <span className="dashboard-subtle-chip">Loading</span> : null}
          </div>

          <div className="mt-5 grid gap-3">
            {members.length === 0 ? (
              <div className="rounded-2xl border border-slate-800/80 bg-slate-950/50 px-4 py-3 text-sm text-slate-400">
                No workspace members found yet.
              </div>
            ) : (
              members.map((member) => (
                <div key={`${member.user_id}-${member.email}`} className="rounded-2xl border border-slate-800/80 bg-slate-950/45 px-4 py-4">
                  <div className="flex flex-col gap-3">
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-slate-900/90 text-sm font-semibold text-cyan-100 ring-1 ring-cyan-400/20">
                        {buildInitials(member.name, member.email)}
                      </div>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-white">{member.name}</p>
                        <p className="truncate text-sm text-slate-400">{member.email}</p>
                      </div>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      {member.is_current ? <span className="dashboard-chip dashboard-chip-success">Current session</span> : null}
                      <span className="dashboard-subtle-chip">{workspaceRoleLabel(member.role)}</span>
                      <span className="dashboard-subtle-chip">Joined {formatDateTime(member.joined_at)}</span>
                    </div>
                    {canManageWorkspaceMember(workspaceRole, isDemo, member.is_current, member.role) ? (
                      <div className="flex flex-col gap-3 border-t border-slate-800/80 pt-3 sm:flex-row sm:items-center sm:justify-between">
                        <div className="flex flex-wrap items-center gap-2">
                          <label className="field-label mb-0">Member Role</label>
                          <select
                            aria-label={`Role for ${member.email}`}
                            className="field-input h-10 min-w-[180px] px-3 py-2"
                            disabled={updateMemberRole.isPending || removeMember.isPending}
                            value={memberRoleDrafts[member.user_id] ?? member.role}
                            onChange={(event) =>
                              setMemberRoleDrafts((current) => ({ ...current, [member.user_id]: event.target.value }))
                            }
                          >
                            {availableRoleOptions.map((role) => (
                              <option key={`${member.user_id}-${role}`} value={role}>
                                {workspaceRoleLabel(role)}
                              </option>
                            ))}
                          </select>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          <Button
                            type="button"
                            variant="secondary"
                            className="h-10 rounded-2xl px-4"
                            disabled={
                              updateMemberRole.isPending ||
                              removeMember.isPending ||
                              (memberRoleDrafts[member.user_id] ?? member.role) === member.role
                            }
                            onClick={() =>
                              updateMemberRole.mutate({
                                userID: member.user_id,
                                role: memberRoleDrafts[member.user_id] ?? member.role,
                              })
                            }
                          >
                            {updateMemberRole.isPending ? "Updating..." : "Update Role"}
                          </Button>
                          <Button
                            type="button"
                            variant="secondary"
                            className="h-10 rounded-2xl border-rose-500/30 bg-rose-500/10 px-4 text-rose-200 hover:bg-rose-500/20"
                            disabled={updateMemberRole.isPending || removeMember.isPending}
                            onClick={() => removeMember.mutate(member.user_id)}
                          >
                            {removeMember.isPending ? "Removing..." : "Remove Member"}
                          </Button>
                        </div>
                      </div>
                    ) : null}
                  </div>
                </div>
              ))
            )}
          </div>
        </Card>

        <div className="grid gap-6">
          <Card className="rounded-[1.5rem]">
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Invitations</p>
            <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Invite Teammate</h3>

            <form
              className="mt-5 grid gap-3"
              onSubmit={(event) => {
                event.preventDefault();
                if (!isDemo) {
                  inviteMember.mutate({ email: inviteValues.email.trim(), role: selectedInviteRole });
                }
              }}
            >
              <label className="grid gap-2">
                <span className="field-label mb-0">Invite Email</span>
                <input
                  aria-label="Invite Email"
                  className="field-input"
                  type="email"
                  disabled={!canManageWorkspaceInvitations || inviteMember.isPending}
                  value={inviteValues.email}
                  onChange={(event) => setInviteValues((current) => ({ ...current, email: event.target.value }))}
                />
              </label>
              <label className="grid gap-2">
                <span className="field-label mb-0">Role</span>
                <select
                  aria-label="Invite Role"
                  className="field-input"
                  disabled={!canManageWorkspaceInvitations || inviteMember.isPending || availableInvitationRoles.length === 0}
                  value={selectedInviteRole}
                  onChange={(event) => setInviteValues((current) => ({ ...current, role: event.target.value }))}
                >
                  {availableInvitationRoles.map((role) => (
                    <option key={`invite-${role}`} value={role}>
                      {workspaceRoleLabel(role)}
                    </option>
                  ))}
                </select>
              </label>
              <div className="flex gap-3">
                <Button
                  type="submit"
                  className="h-11 rounded-2xl px-5"
                  disabled={!canManageWorkspaceInvitations || inviteMember.isPending || inviteValues.email.trim() === "" || availableInvitationRoles.length === 0}
                >
                  {inviteMember.isPending ? "Sending..." : "Send Invite"}
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  className="h-11 rounded-2xl px-5"
                  disabled={inviteMember.isPending}
                  onClick={() => setInviteValues({ email: "", role: "member" })}
                >
                  Clear
                </Button>
              </div>
            </form>
          </Card>

          <Card className="rounded-[1.5rem]">
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Queue</p>
            <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Pending Invitations</h3>
            <div className="mt-5 grid gap-3">
              {invitations.length === 0 ? (
                <div className="rounded-2xl border border-slate-800/80 bg-slate-950/50 px-4 py-3 text-sm text-slate-400">
                  No invitations have been issued for this workspace.
                </div>
              ) : (
                invitations.map((invitation) => (
                  <div key={invitation.id} className="rounded-2xl border border-slate-800/80 bg-slate-950/45 px-4 py-4">
                    <div className="flex flex-col gap-2">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <p className="break-all text-sm font-semibold text-white">{invitation.email}</p>
                        <span className={invitationTone(invitation.status)}>{invitation.status}</span>
                      </div>
                      <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
                        <span className="dashboard-subtle-chip">{workspaceRoleLabel(invitation.role)}</span>
                        <span className="dashboard-subtle-chip">Expires {formatDateTime(invitation.expires_at)}</span>
                      </div>
                      <p className="text-xs text-slate-500">
                        Invited by {invitation.inviter_name} on {formatDateTime(invitation.created_at)}
                      </p>
                      {invitation.status === "pending" && canManageWorkspaceInvitations ? (
                        <div className="border-t border-slate-800/80 pt-3">
                          <Button
                            type="button"
                            variant="secondary"
                            className="h-10 rounded-2xl border-rose-500/30 bg-rose-500/10 px-4 text-rose-200 hover:bg-rose-500/20"
                            disabled={revokeInvitation.isPending}
                            onClick={() => revokeInvitation.mutate(invitation.id)}
                          >
                            {revokeInvitation.isPending ? "Revoking..." : "Revoke Invite"}
                          </Button>
                        </div>
                      ) : null}
                    </div>
                  </div>
                ))
              )}
            </div>
          </Card>
        </div>
      </section>
    </div>
  );
}
