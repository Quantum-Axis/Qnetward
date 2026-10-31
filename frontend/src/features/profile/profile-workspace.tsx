"use client";

import Link from "next/link";
import { useState, useSyncExternalStore } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import {
  AlertRule,
  AuthMessageResponse,
  RecentHistoryTarget,
  RuntimeHealthDependency,
  RuntimeHealthResponse,
  SessionResponse,
  WorkspaceInvitationsResponse,
  WorkspaceTeamResponse,
} from "@/lib/types";
import { formatDateTime } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

function subscribeToOrigin() {
  return () => {};
}

function getOriginSnapshot() {
  return typeof window === "undefined" ? "-" : window.location.origin;
}

function getOriginServerSnapshot() {
  return "-";
}

function buildInitials(name?: string | null, email?: string | null) {
  const source = (name || email || "NW").trim();
  const parts = source.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
  }
  return source.slice(0, 2).toUpperCase();
}

function sessionModeLabel(session?: SessionResponse) {
  if (!session?.authenticated) {
    return "Locked";
  }
  return session.mode === "demo" ? "Demo Session" : "Authenticated";
}

function fallbackAccountRole(session?: SessionResponse) {
  if (!session?.authenticated) {
    return "Locked";
  }
  return session.mode === "demo" ? "Demo Contributor" : "Workspace Admin";
}

function accessLabel(session?: SessionResponse) {
  if (!session?.authenticated) {
    return "No access";
  }
  return session.read_only ? "Read only" : "Full access";
}

function accessTone(session?: SessionResponse) {
  if (!session?.authenticated) {
    return "dashboard-chip dashboard-chip-danger";
  }
  return session.read_only ? "dashboard-chip dashboard-chip-muted" : "dashboard-chip dashboard-chip-success";
}

function dependencyLabel(dependency?: RuntimeHealthDependency) {
  if (!dependency?.enabled) {
    return "Not configured";
  }
  return dependency.healthy ? "Healthy" : "Degraded";
}

function dependencyTone(dependency?: RuntimeHealthDependency) {
  if (!dependency?.enabled) {
    return "text-slate-400";
  }
  return dependency.healthy ? "text-emerald-300" : "text-rose-300";
}

function latestActivity(targets: RecentHistoryTarget[] | undefined) {
  if (!targets || targets.length === 0) {
    return null;
  }
  return [...targets].sort((a, b) => b.last_sample_at - a.last_sample_at)[0];
}

function notificationState(verified: boolean, enabledRules: number) {
  if (!verified) {
    return "Verification pending";
  }
  return enabledRules > 0 ? "Alert delivery active" : "No active alert rules";
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

export function ProfileWorkspace() {
  const origin = useSyncExternalStore(subscribeToOrigin, getOriginSnapshot, getOriginServerSnapshot);
  const queryClient = useQueryClient();
  const [draftValues, setDraftValues] = useState<{ name: string; email: string } | null>(null);
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

  const incomingInvitationsQuery = useQuery({
    queryKey: ["incoming-workspace-invitations"],
    queryFn: () => apiRequest<WorkspaceInvitationsResponse>("/api/workspace/invitations"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const runtimeHealthQuery = useQuery({
    queryKey: ["runtime-health"],
    queryFn: () => apiRequest<RuntimeHealthResponse>("/api/healthz"),
  });

  const recentTargetsQuery = useQuery({
    queryKey: ["recent-history-targets", 6],
    queryFn: () => apiRequest<RecentHistoryTarget[]>("/api/history/recent-targets?limit=6"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const alertRulesQuery = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => apiRequest<AlertRule[]>("/api/alerts/rules"),
    enabled: sessionQuery.data?.authenticated === true,
  });

  const sendResetEmail = useMutation({
    mutationFn: (email: string) =>
      apiRequest<AuthMessageResponse>("/api/auth/forgot-password", {
        method: "POST",
        body: JSON.stringify({ email }),
      }),
  });

  const resendVerification = useMutation({
    mutationFn: (email: string) =>
      apiRequest<AuthMessageResponse>("/api/auth/resend-verification", {
        method: "POST",
        body: JSON.stringify({ email }),
      }),
  });

  const updateProfile = useMutation({
    mutationFn: (values: { name: string; email: string }) =>
      apiRequest<AuthMessageResponse>("/api/auth/profile", {
        method: "PATCH",
        body: JSON.stringify(values),
      }),
    onSuccess: async (response) => {
      if (response.user) {
        setDraftValues({ name: response.user.name, email: response.user.email });
        queryClient.setQueryData<SessionResponse>(["session"], (current) => ({
          authenticated: true,
          user: response.user,
          mode: current?.mode ?? "authenticated",
          read_only: current?.read_only,
        }));
      }
      await queryClient.invalidateQueries({ queryKey: ["session"] });
      await queryClient.invalidateQueries({ queryKey: ["workspace-team"] });
    },
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

  const acceptInvitation = useMutation({
    mutationFn: (invitationID: number) =>
      apiRequest<AuthMessageResponse>("/api/workspace/invitations", {
        method: "POST",
        body: JSON.stringify({ invitation_id: invitationID, action: "accept" }),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["incoming-workspace-invitations"] });
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

  const session = sessionQuery.data;
  const user = session?.user;
  const workspace = workspaceTeamQuery.data?.workspace;
  const members = workspaceTeamQuery.data?.members ?? [];
  const invitations = workspaceTeamQuery.data?.invitations ?? [];
  const incomingInvitations = incomingInvitationsQuery.data?.invitations ?? [];
  const verified = Boolean(user?.email_verified_at);
  const isDemo = session?.mode === "demo";
  const initials = buildInitials(user?.name, user?.email);
  const alertRules = alertRulesQuery.data ?? [];
  const recentTargets = recentTargetsQuery.data ?? [];
  const latestTarget = latestActivity(recentTargets);
  const enabledRules = alertRules.filter((rule) => rule.enabled).length;
  const pingTargets = recentTargets.filter((target) => target.protocol === "ping").length;
  const tcpTargets = recentTargets.filter((target) => target.protocol === "tcpping").length;
  const runtimeHealth = runtimeHealthQuery.data;
  const formValues = draftValues ?? { name: user?.name ?? "", email: user?.email ?? "" };
  const trimmedName = formValues.name.trim();
  const trimmedEmail = formValues.email.trim();
  const trimmedInviteEmail = inviteValues.email.trim();
  const profileChanged = trimmedName !== (user?.name ?? "") || trimmedEmail !== (user?.email ?? "");
  const workspaceRole = workspace?.role ?? fallbackAccountRole(session);
  const pendingInvitations = invitations.filter((invitation) => invitation.status === "pending");
  const availableRoleOptions = memberRoleOptions(workspaceRole);
  const availableInvitationRoles = invitationRoleOptions(workspaceRole);
  const selectedInviteRole = availableInvitationRoles.includes(inviteValues.role) ? inviteValues.role : (availableInvitationRoles[0] ?? "member");
  const canManageWorkspaceInvitations = canManageInvitations(workspaceRole, isDemo);

  function updateField(field: "name" | "email", value: string) {
    if (updateProfile.isError || updateProfile.isSuccess) {
      updateProfile.reset();
    }
    setDraftValues((current) => ({
      ...(current ?? { name: user?.name ?? "", email: user?.email ?? "" }),
      [field]: value,
    }));
  }

  function updateInviteField(field: "email" | "role", value: string) {
    if (inviteMember.isError || inviteMember.isSuccess) {
      inviteMember.reset();
    }
    setInviteValues((current) => ({ ...current, [field]: value }));
  }

  function updateMemberRoleDraft(userID: number, role: string) {
    if (updateMemberRole.isError || updateMemberRole.isSuccess) {
      updateMemberRole.reset();
    }
    setMemberRoleDrafts((current) => ({ ...current, [userID]: role }));
  }

  if (sessionQuery.isLoading) {
    return <div className="dashboard-panel rounded-[1.5rem] p-6 text-sm text-slate-400">Loading account settings.</div>;
  }

  if (sessionQuery.isError) {
    return (
      <div className="rounded-[1.5rem] border border-rose-500/30 bg-rose-500/10 px-5 py-6 text-sm text-rose-200">
        {(sessionQuery.error as ApiError).message || "Unable to load account settings."}
      </div>
    );
  }

  return (
    <div className="grid gap-6">
      <section className="dashboard-hero-panel rounded-[1.7rem] p-5 md:p-6">
        <div className="flex flex-col gap-6 xl:flex-row xl:items-start xl:justify-between">
          <div className="flex min-w-0 items-start gap-4">
            <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-[1.35rem] bg-[linear-gradient(135deg,rgba(68,212,200,0.96),rgba(56,189,248,0.92),rgba(245,158,11,0.88))] text-lg font-bold text-slate-950 shadow-[0_18px_40px_rgba(2,6,12,0.26)]">
              {initials}
            </div>
            <div className="min-w-0">
              <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Account Settings</p>
              <h2 className="mt-1.5 font-space text-[1.9rem] font-bold leading-tight text-white">{user?.name || "Qnetward Operator"}</h2>
              <p className="mt-1 max-w-3xl break-all text-sm text-slate-400">
                Manage identity, security, team access, workspace membership, and SaaS account operations from one place.
              </p>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <span className={verified ? "dashboard-chip dashboard-chip-success" : "dashboard-chip dashboard-chip-danger"}>
                  {verified ? "Email Verified" : "Verification Pending"}
                </span>
                <span className={accessTone(session)}>{accessLabel(session)}</span>
                <span className="dashboard-subtle-chip">{workspaceRole}</span>
                <span className="dashboard-subtle-chip">{workspace?.name ?? "Workspace loading"}</span>
                <span className="dashboard-subtle-chip">Member since {user?.created_at ? formatDateTime(user.created_at) : "-"}</span>
              </div>
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2 xl:min-w-[340px]">
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspace Role</p>
              <p className="mt-2 text-lg font-semibold text-white">{workspaceRoleLabel(workspaceRole)}</p>
              <p className="mt-1.5 text-sm text-slate-400">Resolved from the active workspace membership record.</p>
            </div>
            <div className="dashboard-panel-muted rounded-2xl p-4">
              <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Notification State</p>
              <p className="mt-2 text-lg font-semibold text-white">{notificationState(verified, enabledRules)}</p>
              <p className="mt-1.5 text-sm text-slate-400">Primary channel: {user?.email || "-"}</p>
            </div>
          </div>
        </div>
      </section>

      {isDemo ? (
        <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
          Demo mode is active. Personal edits, workspace invitations, billing actions, and delivery changes are disabled for the shared contributor session.
        </div>
      ) : null}

      {sendResetEmail.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(sendResetEmail.error as ApiError).message || "Unable to send reset email."}
        </div>
      ) : null}
      {sendResetEmail.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {sendResetEmail.data.message || "Password reset email issued."}
        </div>
      ) : null}
      {resendVerification.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(resendVerification.error as ApiError).message || "Unable to issue a new verification code."}
        </div>
      ) : null}
      {resendVerification.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {resendVerification.data.message || "A new verification code has been issued."}
          {resendVerification.data.preview_code ? ` OTP: ${resendVerification.data.preview_code}` : ""}
        </div>
      ) : null}
      {updateProfile.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(updateProfile.error as ApiError).message || "Unable to update profile."}
        </div>
      ) : null}
      {updateProfile.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {updateProfile.data.message}
          {updateProfile.data.preview_code ? ` OTP: ${updateProfile.data.preview_code}` : ""}
        </div>
      ) : null}
      {inviteMember.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(inviteMember.error as ApiError).message || "Unable to create workspace invitation."}
        </div>
      ) : null}
      {inviteMember.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {inviteMember.data.message || "Workspace invitation queued."}
        </div>
      ) : null}
      {acceptInvitation.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(acceptInvitation.error as ApiError).message || "Unable to accept workspace invitation."}
        </div>
      ) : null}
      {acceptInvitation.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {acceptInvitation.data.message || "Workspace invitation accepted."}
        </div>
      ) : null}
      {updateMemberRole.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(updateMemberRole.error as ApiError).message || "Unable to update workspace member role."}
        </div>
      ) : null}
      {updateMemberRole.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {updateMemberRole.data.message || "Workspace member role updated."}
        </div>
      ) : null}
      {removeMember.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(removeMember.error as ApiError).message || "Unable to remove workspace member."}
        </div>
      ) : null}
      {removeMember.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {removeMember.data.message || "Workspace member removed."}
        </div>
      ) : null}
      {revokeInvitation.isError ? (
        <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
          {(revokeInvitation.error as ApiError).message || "Unable to revoke workspace invitation."}
        </div>
      ) : null}
      {revokeInvitation.isSuccess ? (
        <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {revokeInvitation.data.message || "Workspace invitation revoked."}
        </div>
      ) : null}

      <section className="grid gap-6 2xl:grid-cols-[minmax(0,1.22fr)_minmax(360px,0.78fr)]">
        <div className="grid gap-6">
          <Card className="rounded-[1.5rem]">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Profile</p>
                <h3 className="mt-1.5 font-space text-[1.65rem] font-bold text-white">Personal Info</h3>
              </div>
              <span className="dashboard-subtle-chip">{sessionModeLabel(session)}</span>
            </div>

            <form
              className="mt-5 grid gap-4"
              onSubmit={(event) => {
                event.preventDefault();
                if (!isDemo) {
                  updateProfile.mutate({ name: trimmedName, email: trimmedEmail });
                }
              }}
            >
              <div className="grid gap-4 lg:grid-cols-2">
                <label className="grid gap-2">
                  <span className="field-label mb-0">Display Name</span>
                  <input
                    aria-label="Display Name"
                    className="field-input"
                    disabled={isDemo || updateProfile.isPending}
                    value={formValues.name}
                    onChange={(event) => updateField("name", event.target.value)}
                  />
                </label>
                <label className="grid gap-2">
                  <span className="field-label mb-0">Profile Email</span>
                  <input
                    aria-label="Profile Email"
                    className="field-input"
                    disabled={isDemo || updateProfile.isPending}
                    type="email"
                    value={formValues.email}
                    onChange={(event) => updateField("email", event.target.value)}
                  />
                </label>
              </div>

              <div className="dashboard-note-panel p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Email Change Policy</p>
                <p className="mt-2 text-sm text-slate-300">
                  Changing the email address keeps the active session, but the account returns to a pending verification state until the new OTP is completed.
                </p>
              </div>

              <div className="flex flex-wrap gap-3">
                <Button
                  type="submit"
                  className="h-11 rounded-2xl px-5"
                  disabled={isDemo || updateProfile.isPending || trimmedName === "" || trimmedEmail === "" || !profileChanged}
                >
                  {updateProfile.isPending ? "Saving..." : "Save Changes"}
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  className="h-11 rounded-2xl px-5"
                  disabled={updateProfile.isPending || !profileChanged}
                  onClick={() => {
                    updateProfile.reset();
                    setDraftValues(null);
                  }}
                >
                  Reset Form
                </Button>
              </div>
            </form>
          </Card>

          <Card className="rounded-[1.5rem]">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Security</p>
                <h3 className="mt-1.5 font-space text-[1.65rem] font-bold text-white">Security & Access</h3>
              </div>
              <span className={accessTone(session)}>{accessLabel(session)}</span>
            </div>

            <div className="mt-5 grid gap-3 sm:grid-cols-2">
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Verification State</p>
                <p className="mt-2 text-lg font-semibold text-white">{verified ? "Verified" : "Pending"}</p>
                <p className="mt-1.5 text-sm text-slate-400">
                  {verified
                    ? `Verified on ${formatDateTime(user?.email_verified_at ?? "")}`
                    : "This email must be verified before the workspace can be treated as fully trusted."}
                </p>
              </div>
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Session Mode</p>
                <p className="mt-2 text-lg font-semibold text-white">{sessionModeLabel(session)}</p>
                <p className="mt-1.5 text-sm text-slate-400">
                  {isDemo ? "Shared contributor access is active." : "This account is signed in with a normal SaaS session."}
                </p>
              </div>
              <div className="dashboard-panel-muted rounded-2xl p-4 sm:col-span-2">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Recovery Actions</p>
                <div className="mt-3 flex flex-wrap gap-3">
                  {!isDemo && !verified && user?.email ? (
                    <Button
                      type="button"
                      variant="secondary"
                      className="h-11 rounded-2xl px-5"
                      disabled={resendVerification.isPending}
                      onClick={() => resendVerification.mutate(user.email)}
                    >
                      {resendVerification.isPending ? "Sending..." : "Resend Verification"}
                    </Button>
                  ) : null}
                  <Button
                    type="button"
                    variant="secondary"
                    className="h-11 rounded-2xl px-5"
                    disabled={!user?.email || isDemo || sendResetEmail.isPending}
                    onClick={() => {
                      if (user?.email && !isDemo) {
                        sendResetEmail.mutate(user.email);
                      }
                    }}
                  >
                    {sendResetEmail.isPending ? "Sending..." : "Send Reset Email"}
                  </Button>
                </div>
              </div>
            </div>
          </Card>

          <Card className="rounded-[1.5rem]">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Notifications</p>
                <h3 className="mt-1.5 font-space text-[1.65rem] font-bold text-white">Notification Delivery</h3>
              </div>
              <Link href="/alerts" className="dashboard-subtle-chip">
                Open Alerts
              </Link>
            </div>

            <div className="mt-5 grid gap-3 sm:grid-cols-3">
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Primary Channel</p>
                <p className="mt-2 break-all text-lg font-semibold text-white">{user?.email || "-"}</p>
                <p className="mt-1.5 text-sm text-slate-400">Default destination for verification, recovery, and alert delivery.</p>
              </div>
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Alert Rules</p>
                <p className="mt-2 text-lg font-semibold text-white">{enabledRules}</p>
                <p className="mt-1.5 text-sm text-slate-400">Enabled rules currently targeting this account.</p>
              </div>
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Latest Monitored Target</p>
                <p className="mt-2 text-lg font-semibold text-white">{latestTarget?.target || "No telemetry yet"}</p>
                <p className="mt-1.5 text-sm text-slate-400">
                  {latestTarget ? formatDateTime(latestTarget.last_sample_at) : "Run a probe to establish recent activity."}
                </p>
              </div>
            </div>
          </Card>
        </div>

        <div className="grid gap-6">
          <Card className="rounded-[1.5rem]">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Workspace</p>
                <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Workspace & Team</h3>
              </div>
              <span className="dashboard-subtle-chip">{members.length} active members</span>
            </div>

            {workspaceTeamQuery.isError ? (
              <div className="mt-5 rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
                {(workspaceTeamQuery.error as ApiError).message || "Unable to load workspace context."}
              </div>
            ) : (
              <div className="mt-5 grid gap-3">
                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="dashboard-panel-muted rounded-2xl p-4">
                    <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspace Name</p>
                    <p className="mt-2 text-lg font-semibold text-white">{workspace?.name ?? "Workspace loading"}</p>
                    <p className="mt-1.5 text-sm text-slate-400">Current SaaS workspace bound to this session.</p>
                  </div>
                  <div className="dashboard-panel-muted rounded-2xl p-4">
                    <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Role</p>
                    <p className="mt-2 text-lg font-semibold text-white">{workspaceRoleLabel(workspaceRole)}</p>
                    <p className="mt-1.5 text-sm text-slate-400">Workspace permission level for this operator.</p>
                  </div>
                  <div className="dashboard-panel-muted rounded-2xl p-4">
                    <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Pending Invites</p>
                    <p className="mt-2 text-lg font-semibold text-white">{pendingInvitations.length}</p>
                    <p className="mt-1.5 text-sm text-slate-400">Queued invitations awaiting acceptance.</p>
                  </div>
                </div>

                <div className="dashboard-panel-muted rounded-2xl p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Team Members</p>
                      <p className="mt-1.5 text-sm text-slate-400">Live roster for the active workspace membership set.</p>
                    </div>
                    {workspaceTeamQuery.isLoading ? <span className="dashboard-subtle-chip">Loading</span> : null}
                  </div>
                  <div className="mt-4 grid gap-3">
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
                                  onChange={(event) => updateMemberRoleDraft(member.user_id, event.target.value)}
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
                      ))
                    )}
                  </div>
                </div>

                <div className="grid gap-3 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
                  <div className="dashboard-panel-muted rounded-2xl p-4">
                    <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Invite Teammate</p>
                    <p className="mt-1.5 text-sm text-slate-400">
                      Send a pending invitation into this workspace. Roles currently supported: owner, admin, member, viewer.
                    </p>

                    <form
                      className="mt-4 grid gap-3"
                      onSubmit={(event) => {
                        event.preventDefault();
                        if (!isDemo) {
                          inviteMember.mutate({ email: trimmedInviteEmail, role: selectedInviteRole });
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
                          onChange={(event) => updateInviteField("email", event.target.value)}
                        />
                      </label>
                      <label className="grid gap-2">
                        <span className="field-label mb-0">Role</span>
                        <select
                          aria-label="Invite Role"
                          className="field-input"
                          disabled={!canManageWorkspaceInvitations || inviteMember.isPending || availableInvitationRoles.length === 0}
                          value={selectedInviteRole}
                          onChange={(event) => updateInviteField("role", event.target.value)}
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
                          disabled={!canManageWorkspaceInvitations || inviteMember.isPending || trimmedInviteEmail === "" || availableInvitationRoles.length === 0}
                        >
                          {inviteMember.isPending ? "Sending..." : "Send Invite"}
                        </Button>
                        <Button
                          type="button"
                          variant="secondary"
                          className="h-11 rounded-2xl px-5"
                          disabled={inviteMember.isPending || (inviteValues.email === "" && inviteValues.role === "member")}
                          onClick={() => {
                            inviteMember.reset();
                            setInviteValues({ email: "", role: "member" });
                          }}
                        >
                          Clear
                        </Button>
                      </div>
                    </form>
                  </div>

                  <div className="dashboard-panel-muted rounded-2xl p-4">
                    <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Pending Invitations</p>
                    <p className="mt-1.5 text-sm text-slate-400">Issued invites are tracked here until they expire or are accepted.</p>
                    <div className="mt-4 grid gap-3">
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
                              <p className="text-xs text-slate-500">Invited by {invitation.inviter_name} on {formatDateTime(invitation.created_at)}</p>
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
                  </div>
                </div>
              </div>
            )}
          </Card>

          <Card className="rounded-[1.5rem]">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Invitations</p>
                <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Incoming Workspace Invites</h3>
              </div>
              <span className="dashboard-subtle-chip">{incomingInvitations.length} open</span>
            </div>

            {incomingInvitationsQuery.isError ? (
              <div className="mt-5 rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
                {(incomingInvitationsQuery.error as ApiError).message || "Unable to load incoming workspace invitations."}
              </div>
            ) : (
              <div className="mt-5 grid gap-3">
                {incomingInvitations.length === 0 ? (
                  <div className="rounded-2xl border border-slate-800/80 bg-slate-950/50 px-4 py-3 text-sm text-slate-400">
                    No incoming workspace invitations for this account.
                  </div>
                ) : (
                  incomingInvitations.map((invitation) => (
                    <div key={`incoming-${invitation.id}`} className="rounded-2xl border border-slate-800/80 bg-slate-950/45 px-4 py-4">
                      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                        <div className="min-w-0">
                          <p className="truncate text-sm font-semibold text-white">{invitation.workspace_name || "Workspace"}</p>
                          <p className="mt-1 text-sm text-slate-400">
                            {workspaceRoleLabel(invitation.role)} invited by {invitation.inviter_name}
                          </p>
                          <p className="mt-2 text-xs text-slate-500">
                            Expires {formatDateTime(invitation.expires_at)} - Issued {formatDateTime(invitation.created_at)}
                          </p>
                        </div>
                        <div className="flex flex-wrap items-center gap-2">
                          <span className={invitationTone(invitation.status)}>{invitation.status}</span>
                          <Button
                            type="button"
                            className="h-10 rounded-2xl px-4"
                            disabled={isDemo || invitation.status !== "pending" || acceptInvitation.isPending}
                            onClick={() => acceptInvitation.mutate(invitation.id)}
                          >
                            {acceptInvitation.isPending ? "Joining..." : "Join Workspace"}
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </Card>

          <Card className="rounded-[1.5rem]">
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Plan & Billing</p>
            <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Subscription & Usage</h3>

            <div className="mt-5 grid gap-3">
              <div className="dashboard-panel-muted rounded-2xl p-4">
                <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Billing State</p>
                <p className="mt-2 text-lg font-semibold text-white">{workspace?.billing_status ?? "trial"}</p>
                <p className="mt-1.5 text-sm text-slate-400">Current subscription billing state for the active workspace.</p>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="dashboard-panel-muted rounded-2xl p-4">
                  <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Tracked Targets</p>
                  <p className="mt-2 text-lg font-semibold text-white">{recentTargets.length}</p>
                  <p className="mt-1.5 text-sm text-slate-400">{pingTargets} ICMP / {tcpTargets} TCP</p>
                </div>
                <div className="dashboard-panel-muted rounded-2xl p-4">
                  <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Current Plan</p>
                  <p className="mt-2 text-lg font-semibold text-white">{workspace?.plan ?? "starter"}</p>
                  <p className="mt-1.5 text-sm text-slate-400">{members.length} members, {alertRules.length} rules configured</p>
                </div>
              </div>
            </div>
          </Card>

          <Card className="rounded-[1.5rem]">
            <p className="text-[0.68rem] uppercase tracking-[0.24em] text-slate-500">Platform Health</p>
            <h3 className="mt-1.5 font-space text-[1.55rem] font-bold text-white">Runtime Health</h3>

            {runtimeHealthQuery.isError ? (
              <div className="mt-5 rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
                {(runtimeHealthQuery.error as ApiError).message || "Unable to load runtime health."}
              </div>
            ) : (
              <div className="mt-5 grid gap-3">
                {[
                  { label: "Database", value: dependencyLabel(runtimeHealth?.database), tone: dependencyTone(runtimeHealth?.database) },
                  { label: "Redis Cache", value: dependencyLabel(runtimeHealth?.redis), tone: dependencyTone(runtimeHealth?.redis) },
                  { label: "Object Storage", value: dependencyLabel(runtimeHealth?.object_storage), tone: dependencyTone(runtimeHealth?.object_storage) },
                ].map((item) => (
                  <div key={item.label} className="dashboard-panel-muted rounded-2xl p-4">
                    <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">{item.label}</p>
                    <p className={`mt-2 text-lg font-semibold ${item.tone}`}>{item.value}</p>
                  </div>
                ))}
                <div className="dashboard-note-panel p-4">
                  <p className="text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Workspace Origin</p>
                  <p className="mt-2 break-all text-sm text-slate-300">{origin}</p>
                  <p className="mt-3 text-[0.68rem] uppercase tracking-[0.2em] text-slate-500">Last Health Check</p>
                  <p className="mt-2 text-sm text-slate-300">
                    {runtimeHealth?.time ? formatDateTime(runtimeHealth.time) : "Waiting for runtime health response."}
                  </p>
                </div>
              </div>
            )}
          </Card>
        </div>
      </section>
    </div>
  );
}
