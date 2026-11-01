"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiRequest, ApiError } from "@/lib/api";
import type { AdminWorkspacesResponse } from "@/lib/types";
import { PageHeader, Panel, StatStrip } from "@/components/tools/probe-ui";
import { initialsFor } from "@/components/settings/settings-ui";
import { HeadRow, Pill, SearchBox, Segmented, TableShell, Td, Th, timeAgo } from "@/features/admin/admin-ui";

type Filter = "all" | "shared" | "solo" | "invites";

function planLabel(plan: string) {
  return plan === "starter" || !plan ? "Free" : plan[0].toUpperCase() + plan.slice(1);
}

export function AdminWorkspacesWorkspace() {
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const workspacesQuery = useQuery({ queryKey: ["admin-workspaces"], queryFn: () => apiRequest<AdminWorkspacesResponse>("/api/admin/workspaces?limit=100") });

  const workspaces = useMemo(() => workspacesQuery.data?.workspaces ?? [], [workspacesQuery.data]);
  const counts = {
    all: workspaces.length,
    shared: workspaces.filter((w) => w.member_count > 1).length,
    solo: workspaces.filter((w) => w.member_count <= 1).length,
    invites: workspaces.filter((w) => w.pending_invitation_count > 0).length,
  };
  const totalMembers = workspaces.reduce((sum, w) => sum + w.member_count, 0);
  const pendingInvites = workspaces.reduce((sum, w) => sum + w.pending_invitation_count, 0);
  const q = query.trim().toLowerCase();
  const rows = workspaces
    .filter((w) => (filter === "shared" ? w.member_count > 1 : filter === "solo" ? w.member_count <= 1 : filter === "invites" ? w.pending_invitation_count > 0 : true))
    .filter((w) => !q || `${w.name} ${w.owner_name} ${w.owner_email}`.toLowerCase().includes(q))
    .sort((a, b) => b.member_count - a.member_count || b.created_at.localeCompare(a.created_at));

  return (
    <div className="grid grid-cols-1 gap-5 [&>*]:min-w-0">
      <PageHeader title="Workspaces" description="Every workspace, largest first. Each one is a tenant boundary for probes, history, and alerts. Showing up to 100." />

      <StatStrip
        items={[
          { label: "Workspaces", value: String(counts.all || "—") },
          { label: "Shared", value: String(counts.shared) },
          { label: "Single member", value: String(counts.solo) },
          { label: "Memberships", value: String(totalMembers) },
          { label: "Pending invites", value: String(pendingInvites), tone: pendingInvites ? "warn" : undefined },
          { label: "Avg. members", value: counts.all ? (totalMembers / counts.all).toFixed(1) : "—" },
        ]}
      />

      <Panel
        title="All workspaces"
        bodyClassName="p-0"
        aside={
          <div className="flex flex-wrap items-center gap-2">
            <Segmented
              label="Filter workspaces"
              value={filter}
              onChange={setFilter}
              options={[
                { id: "all", label: "All", count: counts.all },
                { id: "shared", label: "Shared", count: counts.shared },
                { id: "solo", label: "Solo", count: counts.solo },
                { id: "invites", label: "With invites", count: counts.invites },
              ]}
            />
            <SearchBox value={query} onChange={setQuery} placeholder="Search name or owner" />
          </div>
        }
      >
        {workspacesQuery.isError ? (
          <p className="px-4 pb-4 text-[13px] text-[#ff8fb3]">{(workspacesQuery.error as ApiError).message || "Couldn't load workspaces."}</p>
        ) : workspacesQuery.isLoading ? (
          <p className="px-4 pb-4 text-[13px] text-[var(--lp-subtle)]">Loading workspaces…</p>
        ) : rows.length === 0 ? (
          <p className="px-4 pb-4 text-[13px] text-[var(--lp-subtle)]">No workspaces match.</p>
        ) : (
          <TableShell minWidth={860}>
            <HeadRow>
              <Th>Workspace</Th>
              <Th>Owner</Th>
              <Th>Plan</Th>
              <Th align="right">Members</Th>
              <Th align="right">Invites</Th>
              <Th align="right">Sessions</Th>
              <Th align="right">Created</Th>
            </HeadRow>
            <tbody>
              {rows.map((w) => (
                <tr key={w.id} className="border-b border-[var(--lp-border)] last:border-0 hover:bg-white/[0.015]">
                  <Td>
                    <div className="flex items-center gap-3">
                      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-white/[0.06] text-[11px]" aria-hidden="true">
                        {initialsFor(w.name)}
                      </span>
                      <span className="truncate">{w.name}</span>
                    </div>
                  </Td>
                  <Td>
                    <p className="truncate">{w.owner_name || "—"}</p>
                    <p className="lp-mono truncate text-[12px] text-[var(--lp-subtle)]">{w.owner_email}</p>
                  </Td>
                  <Td>
                    <Pill>{planLabel(w.plan)}</Pill>
                  </Td>
                  <Td align="right" className="lp-mono">
                    {w.member_count}
                  </Td>
                  <Td align="right" className="lp-mono">
                    {w.pending_invitation_count ? <span className="text-[#fcd34d]">{w.pending_invitation_count}</span> : <span className="text-[var(--lp-subtle)]">0</span>}
                  </Td>
                  <Td align="right" className="lp-mono">
                    {w.active_session_count || <span className="text-[var(--lp-subtle)]">0</span>}
                  </Td>
                  <Td align="right" className="text-[var(--lp-subtle)]">
                    <span title={new Date(w.created_at).toLocaleString()}>{timeAgo(w.created_at)}</span>
                  </Td>
                </tr>
              ))}
            </tbody>
          </TableShell>
        )}
      </Panel>
    </div>
  );
}
