"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import type { ComponentType } from "react";
import { BellRing, Mail, Plug, Webhook } from "@/components/icons";
import { DiscordLogo, SlackLogo, TelegramLogo } from "@/components/brand-logos";
import { apiRequest } from "@/lib/api";
import type { AlertRule, ChannelKind, ChannelListResponse, SessionResponse } from "@/lib/types";
import { SettingsCard, SettingsPageTitle } from "@/components/settings/settings-ui";

// The same logos as Integrations, where these channels are set up.
const CHANNELS: { kind: ChannelKind; label: string; icon: ComponentType<{ className?: string }>; detail: string }[] = [
  { kind: "slack", label: "Slack", icon: SlackLogo, detail: "Post alerts to a channel" },
  { kind: "discord", label: "Discord", icon: DiscordLogo, detail: "Post alerts to a server channel" },
  { kind: "telegram", label: "Telegram", icon: TelegramLogo, detail: "Message a chat or group from your bot" },
  { kind: "webhook", label: "Webhook", icon: Webhook, detail: "POST signed JSON to your own endpoint" },
];

function ChannelRow({ icon: Icon, title, detail, status }: { icon: ComponentType<{ className?: string }>; title: string; detail: string; status: string | null }) {
  return (
    <li className="flex items-center gap-3 px-4 py-3">
      <span className="grid h-9 w-9 shrink-0 place-items-center rounded-md border border-[var(--lp-border-strong)] bg-[#111111]">
        <Icon className="h-4 w-4 text-[var(--lp-muted)]" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm">{title}</p>
        <p className="truncate text-xs text-[var(--lp-subtle)]">{detail}</p>
      </div>
      <span className={`shrink-0 text-sm ${status ? "text-[var(--lp-text)]" : "text-[var(--lp-subtle)]"}`}>{status ?? "Not set up"}</span>
    </li>
  );
}

export function NotificationsSettings() {
  const sessionQuery = useQuery({ queryKey: ["session"], queryFn: () => apiRequest<SessionResponse>("/api/auth/session") });
  const rulesQuery = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => apiRequest<AlertRule[]>("/api/alerts/rules"),
    enabled: sessionQuery.data?.authenticated === true,
  });
  const channelsQuery = useQuery({
    queryKey: ["workspace-channels"],
    queryFn: () => apiRequest<ChannelListResponse>("/api/workspace/channels"),
    enabled: sessionQuery.data?.authenticated === true,
  });
  const channels = channelsQuery.data?.channels ?? [];
  const statusOf = (kind: ChannelKind) => {
    const mine = channels.filter((c) => c.kind === kind);
    if (!mine.length) return null;
    const on = mine.filter((c) => c.enabled).length;
    return on === mine.length ? `${on} connected` : `${on} of ${mine.length} on`;
  };
  const email = sessionQuery.data?.user?.email ?? "";
  const rules = rulesQuery.data ?? [];
  const mine = rules.filter((rule) => rule.enabled && rule.recipient_email.toLowerCase() === email.toLowerCase());
  const elsewhere = rules.filter((rule) => rule.enabled && rule.recipient_email.toLowerCase() !== email.toLowerCase());

  return (
    <div className="grid grid-cols-1 gap-5">
      <SettingsPageTitle title="Notifications" description="Where Qnetward reaches you, and what it sends." />

      <SettingsCard
        id="alerts"
        title="Alert emails"
        description="Each alert rule emails one address when it fires and, if you choose, when it recovers."
        hint={
          rulesQuery.isLoading
            ? "Checking your rules…"
            : elsewhere.length
              ? `${elsewhere.length} other active ${elsewhere.length === 1 ? "rule sends" : "rules send"} to a different address.`
              : null
        }
        action={
          <Link href="/alerts" className="console-btn console-btn-secondary">
            <BellRing className="h-3.5 w-3.5" /> Manage alert rules
          </Link>
        }
      >
        {mine.length ? (
          <ul className="divide-y divide-[var(--lp-border)] rounded-lg border border-[var(--lp-border)]">
            {mine.map((rule) => (
              <li key={rule.id} className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm">
                <span className="min-w-0 truncate">{rule.name}</span>
                <span className="lp-mono shrink-0 text-xs text-[var(--lp-subtle)]">
                  {rule.protocol === "tcpping" && rule.port ? `${rule.target}:${rule.port}` : rule.target}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="rounded-lg border border-dashed border-[var(--lp-border)] px-4 py-5 text-center text-sm text-[var(--lp-subtle)]">
            {rulesQuery.isLoading ? "Loading…" : <>No active rules send to <span className="lp-mono">{email}</span>.</>}
          </p>
        )}
      </SettingsCard>

      <SettingsCard
        id="channels"
        title="Channels"
        description="Account emails always go to your sign-in address. Chat and webhook channels are shared by the workspace and set up in Integrations."
        action={
          <Link href="/workspace/integrations" className="console-btn console-btn-secondary">
            <Plug className="h-3.5 w-3.5" /> Manage in Integrations
          </Link>
        }
      >
        <ul className="divide-y divide-[var(--lp-border)] rounded-lg border border-[var(--lp-border)]">
          <ChannelRow icon={Mail} title="Email" detail={email || "—"} status="On" />
          {CHANNELS.map((c) => (
            <ChannelRow key={c.kind} icon={c.icon} title={c.label} detail={c.detail} status={channelsQuery.isLoading ? "…" : statusOf(c.kind)} />
          ))}
        </ul>
      </SettingsCard>
    </div>
  );
}
