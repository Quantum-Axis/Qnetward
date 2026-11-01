"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { apiRequest, ApiError } from "@/lib/api";
import type { AlertRule, RecentHistoryTarget, SessionResponse, WorkspaceTeamResponse } from "@/lib/types";
import { cn } from "@/lib/utils";
import { Notice, SettingsCard, SettingsPageTitle } from "@/components/settings/settings-ui";

type PlanId = "free" | "pro" | "enterprise";

// Mirrors the public pricing on the landing page (features/landing/pricing.tsx).
const PLANS: { id: PlanId; name: string; price: string; blurb: string; features: string[] }[] = [
  {
    id: "free",
    name: "Free",
    price: "$0 during beta",
    blurb: "Everything that works today, for you and your team.",
    features: ["All four diagnostic tools", "Live results and history", "Email alert rules", "Workspaces, roles, invitations", "CSV and JSON export"],
  },
  {
    id: "pro",
    name: "Pro",
    price: "Coming soon",
    blurb: "Continuous monitoring from more places.",
    features: ["Probe agents for private networks", "Multi-region probing", "Longer history", "Slack, Telegram, Discord, webhooks"],
  },
  {
    id: "enterprise",
    name: "Enterprise",
    price: "Custom",
    blurb: "For ISPs, MSPs, and regulated teams.",
    features: ["SSO and SCIM", "Dedicated or self-hosted", "Audit log export", "Priority support and SLA"],
  },
];

// The workspace record still uses the pre-launch plan names.
function planFor(stored?: string): PlanId {
  if (stored === "pro") return "pro";
  if (stored === "enterprise") return "enterprise";
  return "free";
}

function statusLabel(status?: string) {
  switch (status) {
    case "active":
      return { text: "Active", tone: "good" as const };
    case "past_due":
      return { text: "Payment due", tone: "bad" as const };
    case "canceled":
      return { text: "Canceled", tone: "muted" as const };
    default:
      return { text: "Beta", tone: "muted" as const };
  }
}

function UsageRow({ label, value, detail, href }: { label: string; value: number | string; detail: string; href?: string }) {
  const body = (
    <>
      <div className="min-w-0 flex-1">
        <p className="text-[13px]">{label}</p>
        <p className="text-[12px] text-[var(--lp-subtle)]">{detail}</p>
      </div>
      <span className="lp-mono text-[15px]">{value}</span>
    </>
  );
  return (
    <li>
      {href ? (
        <Link href={href} className="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-white/[0.02]">
          {body}
        </Link>
      ) : (
        <div className="flex items-center gap-3 px-4 py-3">{body}</div>
      )}
    </li>
  );
}

export function BillingWorkspace() {
  const sessionQuery = useQuery({ queryKey: ["session"], queryFn: () => apiRequest<SessionResponse>("/api/auth/session") });
  const authenticated = sessionQuery.data?.authenticated === true;
  const teamQuery = useQuery({
    queryKey: ["workspace-team"],
    queryFn: () => apiRequest<WorkspaceTeamResponse>("/api/workspace/team"),
    enabled: authenticated,
  });
  const targetsQuery = useQuery({
    queryKey: ["recent-history-targets", 50],
    queryFn: () => apiRequest<RecentHistoryTarget[]>("/api/history/recent-targets?limit=50"),
    enabled: authenticated,
  });
  const rulesQuery = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => apiRequest<AlertRule[]>("/api/alerts/rules"),
    enabled: authenticated,
  });

  const workspace = teamQuery.data?.workspace;
  const members = teamQuery.data?.members ?? [];
  const pendingSeats = (teamQuery.data?.invitations ?? []).filter((i) => i.status === "pending").length;
  const targets = targetsQuery.data ?? [];
  const activeRules = (rulesQuery.data ?? []).filter((r) => r.enabled).length;
  const current = planFor(workspace?.plan);
  const status = statusLabel(workspace?.billing_status);
  const plan = PLANS.find((p) => p.id === current) ?? PLANS[0];
  const isOwner = workspace?.role === "owner";

  return (
    <div className="grid grid-cols-1 gap-5">
      <SettingsPageTitle title="Billing and plan" description={`The plan and usage for ${workspace?.name ?? "this workspace"}.`} />

      {teamQuery.isError ? <Notice tone="error">{(teamQuery.error as ApiError).message || "Couldn't load billing details."}</Notice> : null}

      <SettingsCard
        id="plan"
        title="Current plan"
        description="Qnetward is free while in beta. Paid plans and invoices arrive with Pro."
        hint={isOwner ? "You'll be able to change plans here." : "Only workspace owners can change the plan."}
      >
        <div className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-[var(--lp-border)] bg-white/[0.02] p-4">
          <div>
            <p className="flex items-center gap-2 text-[18px] font-medium tracking-[-0.01em]">
              {plan.name}
              <span
                className={cn(
                  "rounded-md border px-1.5 py-px text-[11px] font-normal",
                  status.tone === "good"
                    ? "border-[rgba(74,222,128,0.25)] bg-[rgba(74,222,128,0.08)] text-[#86efac]"
                    : status.tone === "bad"
                      ? "border-[rgba(255,61,127,0.3)] bg-[rgba(255,61,127,0.08)] text-[#ff8fb3]"
                      : "border-[var(--lp-border)] bg-white/[0.03] text-[var(--lp-muted)]",
                )}
              >
                {status.text}
              </span>
            </p>
            <p className="mt-0.5 text-[13px] text-[var(--lp-muted)]">{plan.price}</p>
          </div>
          <p className="max-w-xs text-[12px] leading-relaxed text-[var(--lp-subtle)]">{plan.blurb}</p>
        </div>
      </SettingsCard>

      <SettingsCard id="usage" title="Usage" description="What this workspace uses today. Plan limits will be measured against these.">
        <ul className="divide-y divide-[var(--lp-border)] rounded-lg border border-[var(--lp-border)]">
          <UsageRow
            label="Seats"
            value={members.length}
            detail={pendingSeats ? `${pendingSeats} more invited, not yet joined` : "Members of this workspace"}
            href="/workspace/team"
          />
          <UsageRow
            label="Monitored targets"
            value={targetsQuery.isLoading ? "…" : targets.length >= 50 ? "50+" : targets.length}
            detail="Targets with stored results"
            href="/history"
          />
          <UsageRow label="Active alert rules" value={rulesQuery.isLoading ? "…" : activeRules} detail="Rules checked after every probe" href="/alerts" />
        </ul>
      </SettingsCard>

      <SettingsCard id="plans" title="Plans" description="Pro and Enterprise aren't available to buy yet.">
        <div className="grid gap-3 md:grid-cols-3">
          {PLANS.map((p) => {
            const isCurrent = p.id === current;
            return (
              <div
                key={p.id}
                className={cn(
                  "flex flex-col rounded-lg border p-4",
                  isCurrent ? "border-[rgba(255,61,127,0.45)] bg-[rgba(255,61,127,0.04)]" : "border-[var(--lp-border)]",
                )}
              >
                <div className="flex items-center justify-between">
                  <p className="text-[14px] font-medium">{p.name}</p>
                  {isCurrent ? <span className="rounded border border-[rgba(255,61,127,0.4)] px-1.5 text-[11px] leading-4 text-[#ff8fb3]">Current</span> : null}
                </div>
                <p className="mt-0.5 text-[12px] text-[var(--lp-subtle)]">{p.price}</p>
                <ul className="mt-3 grid gap-1.5 text-[12px] text-[var(--lp-muted)]">
                  {p.features.map((f) => (
                    <li key={f} className="flex items-start gap-1.5">
                      <Check className="mt-0.5 h-3 w-3 shrink-0 text-[var(--lp-subtle)]" />
                      {f}
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
        </div>
      </SettingsCard>

      <SettingsCard id="invoices" title="Invoices" description="Receipts for paid plans will be listed here.">
        <p className="rounded-lg border border-dashed border-[var(--lp-border)] px-4 py-5 text-center text-[13px] text-[var(--lp-subtle)]">No invoices. This workspace is on the free beta.</p>
      </SettingsCard>
    </div>
  );
}
