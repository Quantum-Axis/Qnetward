"use client";

import Link from "next/link";
import { ReactNode, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight, BellRing, Check, CreditCard, KeyRound, Mail, Shield, Users } from "lucide-react";
import { apiRequest, ApiError } from "@/lib/api";
import {
  AlertRule,
  AuthMessageResponse,
  RecentHistoryTarget,
  SessionResponse,
  WorkspaceInvitationsResponse,
  WorkspaceListResponse,
} from "@/lib/types";
import { cn, formatDateTime } from "@/lib/utils";
import { DefinitionList, PageHeader, RunStatus } from "@/components/tools/probe-ui";

function buildInitials(name?: string | null, email?: string | null) {
  const source = (name || email || "QN").trim();
  const parts = source.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0][0] ?? ""}${parts[1][0] ?? ""}`.toUpperCase();
  }
  return source.slice(0, 2).toUpperCase();
}

function roleLabel(role?: string) {
  switch (role) {
    case "owner":
      return "Owner";
    case "admin":
      return "Admin";
    case "viewer":
      return "Viewer";
    case "member":
      return "Member";
    default:
      return role ?? "—";
  }
}

function formatDate(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleDateString([], { year: "numeric", month: "short", day: "numeric" });
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** A settings block: what it is on the left, the controls on the right, actions in a footer bar. */
function Section({
  title,
  description,
  children,
  footer,
  id,
}: {
  title: string;
  description: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  id?: string;
}) {
  return (
    <section id={id} className="rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f]" aria-labelledby={id ? `${id}-title` : undefined}>
      <div className="grid gap-4 p-5 md:grid-cols-[220px_minmax(0,1fr)] md:gap-8">
        <div>
          <h2 id={id ? `${id}-title` : undefined} className="text-[14px] font-medium">
            {title}
          </h2>
          <p className="mt-1 text-[13px] leading-relaxed text-[var(--lp-muted)]">{description}</p>
        </div>
        <div className="min-w-0">{children}</div>
      </div>
      {footer ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-b-xl border-t border-[var(--lp-border)] bg-white/[0.015] px-5 py-3">{footer}</div>
      ) : null}
    </section>
  );
}

function Notice({ tone, children }: { tone: "success" | "error" | "info"; children: ReactNode }) {
  const styles = {
    success: "border-[rgba(74,222,128,0.25)] bg-[rgba(74,222,128,0.08)] text-[#86efac]",
    error: "border-[rgba(255,61,127,0.3)] bg-[rgba(255,61,127,0.08)] text-[#ff8fb3]",
    info: "border-[rgba(251,191,36,0.3)] bg-[rgba(251,191,36,0.08)] text-[#fcd34d]",
  };
  return (
    <p role={tone === "error" ? "alert" : "status"} className={cn("rounded-lg border px-3 py-2.5 text-[13px]", styles[tone])}>
      {children}
    </p>
  );
}

function ShortcutLink({ href, icon: Icon, label, hint }: { href: string; icon: typeof Shield; label: string; hint: string }) {
  return (
    <Link href={href} className="group flex items-center gap-3 rounded-lg px-2 py-2 transition-colors hover:bg-white/[0.04]">
      <span className="grid h-8 w-8 shrink-0 place-items-center rounded-md border border-[var(--lp-border-strong)] bg-white/[0.03]">
        <Icon className="h-3.5 w-3.5 text-[var(--lp-muted)]" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-[13px] text-[var(--lp-text)]">{label}</span>
        <span className="block truncate text-[12px] text-[var(--lp-subtle)]">{hint}</span>
      </span>
      <ArrowUpRight className="h-3.5 w-3.5 text-[var(--lp-subtle)] transition-colors group-hover:text-[var(--lp-text)]" />
    </Link>
  );
}

export function ProfileWorkspace() {
  const queryClient = useQueryClient();
  const [draftValues, setDraftValues] = useState<{ name: string; email: string } | null>(null);

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
  });
  const authenticated = sessionQuery.data?.authenticated === true;

  const workspacesQuery = useQuery({
    queryKey: ["workspaces"],
    queryFn: () => apiRequest<WorkspaceListResponse>("/api/workspaces"),
    enabled: authenticated,
  });

  const incomingInvitationsQuery = useQuery({
    queryKey: ["incoming-workspace-invitations"],
    queryFn: () => apiRequest<WorkspaceInvitationsResponse>("/api/workspace/invitations"),
    enabled: authenticated,
  });

  const recentTargetsQuery = useQuery({
    queryKey: ["recent-history-targets", 6],
    queryFn: () => apiRequest<RecentHistoryTarget[]>("/api/history/recent-targets?limit=6"),
    enabled: authenticated,
  });

  const alertRulesQuery = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => apiRequest<AlertRule[]>("/api/alerts/rules"),
    enabled: authenticated,
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
        setDraftValues(null);
        queryClient.setQueryData<SessionResponse>(["session"], (current) => ({
          ...current,
          authenticated: true,
          user: response.user,
          mode: current?.mode ?? "authenticated",
        }));
      }
      await queryClient.invalidateQueries({ queryKey: ["session"] });
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
      await queryClient.invalidateQueries({ queryKey: ["workspaces"] });
      await queryClient.invalidateQueries({ queryKey: ["workspace-team"] });
    },
  });

  const session = sessionQuery.data;
  const user = session?.user;
  const isDemo = session?.mode === "demo";
  const verified = Boolean(user?.email_verified_at);
  const workspaces = workspacesQuery.data?.workspaces ?? [];
  const activeWorkspaceId = workspacesQuery.data?.active_workspace_id ?? session?.active_workspace_id;
  const activeWorkspace = workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0];
  const incomingInvitations = incomingInvitationsQuery.data?.invitations ?? [];
  const pendingInvitations = incomingInvitations.filter((invitation) => invitation.status === "pending");
  const enabledRules = (alertRulesQuery.data ?? []).filter((rule) => rule.enabled && rule.recipient_email === user?.email).length;
  const latestTarget = [...(recentTargetsQuery.data ?? [])].sort((a, b) => b.last_sample_at - a.last_sample_at)[0];

  const formValues = draftValues ?? { name: user?.name ?? "", email: user?.email ?? "" };
  const trimmedName = formValues.name.trim();
  const trimmedEmail = formValues.email.trim();
  const nameChanged = trimmedName !== (user?.name ?? "");
  const emailChanged = trimmedEmail.toLowerCase() !== (user?.email ?? "").toLowerCase();
  const profileChanged = nameChanged || emailChanged;
  const emailInvalid = trimmedEmail !== "" && !EMAIL_PATTERN.test(trimmedEmail);
  const canSave = !isDemo && profileChanged && trimmedName !== "" && trimmedEmail !== "" && !emailInvalid && !updateProfile.isPending;

  function updateField(field: "name" | "email", value: string) {
    if (updateProfile.isError || updateProfile.isSuccess) {
      updateProfile.reset();
    }
    setDraftValues((current) => ({
      ...(current ?? { name: user?.name ?? "", email: user?.email ?? "" }),
      [field]: value,
    }));
  }

  if (sessionQuery.isLoading) {
    return <p className="text-[13px] text-[var(--lp-subtle)]">Loading your profile…</p>;
  }

  if (sessionQuery.isError) {
    return <Notice tone="error">{(sessionQuery.error as ApiError).message || "Unable to load your profile."}</Notice>;
  }

  return (
    <div className="grid grid-cols-1 gap-5 [&>*]:min-w-0">
      <PageHeader
        title="Profile"
        description="Your name, email, and sign-in details. Team members and billing are managed under Workspace."
        status={
          isDemo ? (
            <RunStatus tone="idle" label="Demo session" />
          ) : verified ? (
            <RunStatus tone="done" label="Email verified" />
          ) : (
            <RunStatus tone="error" label="Email not verified" />
          )
        }
      />

      <section className="relative overflow-hidden rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f] p-5">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0"
          style={{ background: "radial-gradient(ellipse 45% 120% at 0% 0%, rgba(255,61,127,0.08), transparent 70%)" }}
        />
        <div className="relative flex flex-col gap-4 sm:flex-row sm:items-center">
          <div
            className="grid h-16 w-16 shrink-0 place-items-center rounded-2xl text-[20px] font-medium text-white"
            style={{ background: "linear-gradient(135deg, #ff3d7f 0%, #a855f7 100%)" }}
            aria-hidden="true"
          >
            {buildInitials(user?.name, user?.email)}
          </div>
          <div className="min-w-0 flex-1">
            <p className="truncate text-[18px] font-medium tracking-[-0.01em]">{user?.name || "Qnetward user"}</p>
            <p className="lp-mono mt-0.5 truncate text-[13px] text-[var(--lp-muted)]">{user?.email}</p>
            <div className="mt-2.5 flex flex-wrap items-center gap-1.5 text-[12px]">
              {activeWorkspace ? (
                <span className="rounded-md border border-[var(--lp-border)] bg-white/[0.03] px-2 py-0.5 text-[var(--lp-muted)]">
                  {roleLabel(activeWorkspace.role)} · {activeWorkspace.name}
                </span>
              ) : null}
              {session?.platform_role === "admin" ? (
                <span className="rounded-md border border-[rgba(167,139,250,0.3)] bg-[rgba(167,139,250,0.1)] px-2 py-0.5 text-[#c4b5fd]">Platform admin</span>
              ) : null}
              {session?.read_only ? (
                <span className="rounded-md border border-[var(--lp-border)] bg-white/[0.03] px-2 py-0.5 text-[var(--lp-muted)]">Read only</span>
              ) : null}
              <span className="px-1 text-[var(--lp-subtle)]">Joined {formatDate(user?.created_at)}</span>
            </div>
          </div>
        </div>
      </section>

      {isDemo ? (
        <Notice tone="info">This is the shared demo account. Name, email, and password changes are turned off so the next visitor finds it the same way.</Notice>
      ) : null}

      <div className="grid grid-cols-1 items-start gap-5 xl:grid-cols-[minmax(0,1fr)_300px] [&>*]:min-w-0">
        <div className="grid gap-5">
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (canSave) updateProfile.mutate({ name: trimmedName, email: trimmedEmail });
            }}
          >
            <Section
              id="personal"
              title="Personal info"
              description="Shown to teammates in your workspaces and used on alert and invitation emails."
              footer={
                <>
                  <span className="text-[12px] text-[var(--lp-subtle)]">
                    {updateProfile.isSuccess ? (
                      <span className="inline-flex items-center gap-1.5 text-[#86efac]">
                        <Check className="h-3.5 w-3.5" /> {updateProfile.data?.message || "Saved."}
                      </span>
                    ) : profileChanged ? (
                      "You have unsaved changes."
                    ) : (
                      "Up to date."
                    )}
                  </span>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      className="console-btn console-btn-secondary"
                      disabled={!profileChanged || updateProfile.isPending}
                      onClick={() => {
                        updateProfile.reset();
                        setDraftValues(null);
                      }}
                    >
                      Discard
                    </button>
                    <button type="submit" className="console-btn console-btn-primary" disabled={!canSave}>
                      {updateProfile.isPending ? "Saving" : "Save changes"}
                    </button>
                  </div>
                </>
              }
            >
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block">
                  <span className="mb-1.5 block text-[12px] text-[var(--lp-muted)]">Name</span>
                  <input
                    id="profile-name"
                    aria-label="Display Name"
                    className="h-9 w-full px-3"
                    autoComplete="name"
                    maxLength={80}
                    disabled={isDemo || updateProfile.isPending}
                    value={formValues.name}
                    onChange={(event) => updateField("name", event.target.value)}
                  />
                  {trimmedName === "" && draftValues ? <span className="mt-1 block text-[12px] text-[#ff8fb3]">Name can&apos;t be empty.</span> : null}
                </label>
                <label className="block">
                  <span className="mb-1.5 block text-[12px] text-[var(--lp-muted)]">Email</span>
                  <input
                    id="profile-email"
                    aria-label="Profile Email"
                    className="h-9 w-full px-3"
                    type="email"
                    autoComplete="email"
                    disabled={isDemo || updateProfile.isPending}
                    value={formValues.email}
                    onChange={(event) => updateField("email", event.target.value)}
                  />
                  {emailInvalid ? <span className="mt-1 block text-[12px] text-[#ff8fb3]">Enter a valid email address.</span> : null}
                </label>
              </div>
              {emailChanged && !emailInvalid && trimmedEmail ? (
                <p className="mt-3 flex items-start gap-2 text-[12px] leading-relaxed text-[var(--lp-muted)]">
                  <Mail className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[#fcd34d]" />
                  We&apos;ll send a code to {trimmedEmail}. You stay signed in, but the account counts as unverified until you enter it.
                </p>
              ) : null}
              {updateProfile.isError ? (
                <div className="mt-3">
                  <Notice tone="error">{(updateProfile.error as ApiError).message || "Couldn't save your profile."}</Notice>
                </div>
              ) : null}
            </Section>
          </form>

          <Section
            id="verification"
            title="Email verification"
            description="A verified address is required for alert emails and invitations to reach you."
          >
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-[var(--lp-border)] bg-white/[0.02] px-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <span
                  className={cn("grid h-8 w-8 shrink-0 place-items-center rounded-full", verified ? "bg-[rgba(74,222,128,0.12)]" : "bg-[rgba(251,191,36,0.12)]")}
                >
                  {verified ? <Check className="h-4 w-4 text-[#4ade80]" /> : <Mail className="h-4 w-4 text-[#fbbf24]" />}
                </span>
                <div className="min-w-0">
                  <p className="lp-mono truncate text-[13px]">{user?.email}</p>
                  <p className="text-[12px] text-[var(--lp-subtle)]">
                    {verified ? `Verified ${formatDateTime(user?.email_verified_at ?? "")}` : "Not verified yet"}
                  </p>
                </div>
              </div>
              {!verified && !isDemo && user?.email ? (
                <button
                  type="button"
                  className="console-btn console-btn-secondary"
                  disabled={resendVerification.isPending}
                  onClick={() => resendVerification.mutate(user.email)}
                >
                  {resendVerification.isPending ? "Sending" : "Resend code"}
                </button>
              ) : null}
            </div>
            {resendVerification.isSuccess ? (
              <div className="mt-3">
                <Notice tone="success">{resendVerification.data.message || "Verification code sent."}</Notice>
              </div>
            ) : null}
            {resendVerification.isError ? (
              <div className="mt-3">
                <Notice tone="error">{(resendVerification.error as ApiError).message || "Couldn't send the code."}</Notice>
              </div>
            ) : null}
          </Section>

          <Section
            id="password"
            title="Password"
            description={
              <>
                Reset it by email. Active sessions and sign-in details are on the{" "}
                <Link href="/security" className="text-[var(--lp-accent)] hover:underline">
                  Security
                </Link>{" "}
                page.
              </>
            }
          >
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p className="text-[13px] text-[var(--lp-muted)]">
                We&apos;ll email a reset link to <span className="lp-mono text-[var(--lp-text)]">{user?.email}</span>. It expires after 30 minutes.
              </p>
              <button
                type="button"
                className="console-btn console-btn-secondary"
                disabled={!user?.email || isDemo || sendResetEmail.isPending}
                onClick={() => user?.email && !isDemo && sendResetEmail.mutate(user.email)}
              >
                <KeyRound className="h-3.5 w-3.5" /> {sendResetEmail.isPending ? "Sending" : "Send reset email"}
              </button>
            </div>
            {sendResetEmail.isSuccess ? (
              <div className="mt-3">
                <Notice tone="success">{sendResetEmail.data.message || "Check your inbox for the reset link."}</Notice>
              </div>
            ) : null}
            {sendResetEmail.isError ? (
              <div className="mt-3">
                <Notice tone="error">{(sendResetEmail.error as ApiError).message || "Couldn't send the reset email."}</Notice>
              </div>
            ) : null}
          </Section>

          <Section id="notifications" title="Notifications" description="Where Qnetward reaches you. Slack, Telegram, and webhooks come with Integrations.">
            <ul className="divide-y divide-[var(--lp-border)] rounded-lg border border-[var(--lp-border)]">
              <li className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="min-w-0">
                  <p className="text-[13px]">Alert emails</p>
                  <p className="text-[12px] text-[var(--lp-subtle)]">
                    {alertRulesQuery.isLoading ? "Checking your rules…" : `${enabledRules} active ${enabledRules === 1 ? "rule sends" : "rules send"} to ${user?.email}`}
                  </p>
                </div>
                <Link href="/alerts" className="console-btn console-btn-secondary shrink-0">
                  <BellRing className="h-3.5 w-3.5" /> Alert rules
                </Link>
              </li>
              <li className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="min-w-0">
                  <p className="text-[13px]">Account emails</p>
                  <p className="text-[12px] text-[var(--lp-subtle)]">Verification codes, password resets, and invitations. Always on.</p>
                </div>
                <span className="shrink-0 rounded-md border border-[rgba(74,222,128,0.25)] bg-[rgba(74,222,128,0.08)] px-2 py-0.5 text-[12px] text-[#86efac]">On</span>
              </li>
            </ul>
          </Section>

          <Section
            id="invitations"
            title="Invitations"
            description="Workspaces that invited this email address. Joining adds the workspace to your switcher."
          >
            {incomingInvitationsQuery.isError ? (
              <Notice tone="error">{(incomingInvitationsQuery.error as ApiError).message || "Couldn't load invitations."}</Notice>
            ) : incomingInvitations.length === 0 ? (
              <p className="rounded-lg border border-dashed border-[var(--lp-border)] px-4 py-6 text-center text-[13px] text-[var(--lp-subtle)]">No invitations right now.</p>
            ) : (
              <ul className="divide-y divide-[var(--lp-border)] rounded-lg border border-[var(--lp-border)]">
                {incomingInvitations.map((invitation) => {
                  const pending = invitation.status === "pending";
                  return (
                    <li key={invitation.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
                      <div className="min-w-0">
                        <p className="truncate text-[13px]">{invitation.workspace_name || "Workspace"}</p>
                        <p className="text-[12px] text-[var(--lp-subtle)]">
                          {roleLabel(invitation.role)} · invited by {invitation.inviter_name} ·{" "}
                          {pending ? `expires ${formatDate(invitation.expires_at)}` : invitation.status}
                        </p>
                      </div>
                      <button
                        type="button"
                        className="console-btn console-btn-primary"
                        disabled={isDemo || !pending || acceptInvitation.isPending}
                        onClick={() => acceptInvitation.mutate(invitation.id)}
                      >
                        {acceptInvitation.isPending && acceptInvitation.variables === invitation.id ? "Joining" : "Join workspace"}
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
            {acceptInvitation.isError ? (
              <div className="mt-3">
                <Notice tone="error">{(acceptInvitation.error as ApiError).message || "Couldn't join the workspace."}</Notice>
              </div>
            ) : null}
          </Section>
        </div>

        <aside className="grid gap-5 xl:sticky xl:top-20">
          <section className="rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f] p-4">
            <h2 className="mb-3 text-[14px] font-medium">Account</h2>
            <DefinitionList
              items={[
                { label: "Session", value: isDemo ? "Demo" : "Signed in" },
                { label: "Access", value: session?.read_only ? "Read only" : "Full" },
                { label: "Workspaces", value: workspacesQuery.isLoading ? "…" : String(workspaces.length) },
                { label: "Invitations", value: String(pendingInvitations.length) },
                { label: "Last probe", value: latestTarget ? latestTarget.target : "—" },
                { label: "Account ID", value: user?.id ? `#${user.id}` : "—" },
              ]}
            />
          </section>

          <section className="rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f] p-2">
            <h2 className="px-2 pb-1 pt-2 text-[14px] font-medium">Workspaces</h2>
            {workspaces.length ? (
              <ul className="px-2 pb-1">
                {workspaces.map((w) => (
                  <li key={w.id} className="flex items-center justify-between gap-2 py-1.5 text-[13px]">
                    <span className="flex min-w-0 items-center gap-2">
                      <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", w.id === activeWorkspace?.id ? "bg-[#4ade80]" : "bg-white/20")} />
                      <span className="truncate">{w.name}</span>
                    </span>
                    <span className="shrink-0 text-[12px] text-[var(--lp-subtle)]">{roleLabel(w.role)}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="px-2 pb-2 text-[12px] text-[var(--lp-subtle)]">{workspacesQuery.isLoading ? "Loading…" : "No workspaces."}</p>
            )}
            <div className="mt-1 border-t border-[var(--lp-border)] pt-1">
              <ShortcutLink href="/workspace/team" icon={Users} label="Team" hint="Members, roles, and invites" />
              <ShortcutLink href="/workspace/billing" icon={CreditCard} label="Billing" hint="Plan and usage" />
              <ShortcutLink href="/security" icon={Shield} label="Security" hint="Sessions and sign-in" />
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
