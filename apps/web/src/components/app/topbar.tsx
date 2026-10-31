"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Menu, Plus } from "lucide-react";
import { SessionResponse } from "@/lib/types";
import { useShellStore } from "@/store/ui-store";

const titles: Record<string, [section: string, page: string]> = {
  "/overview": ["Diagnostics", "Overview"],
  "/icmp-ping": ["Diagnostics", "ICMP ping"],
  "/tcp-ping": ["Diagnostics", "TCP ping"],
  "/port-scan": ["Diagnostics", "Port scan"],
  "/dns-lookup": ["Diagnostics", "DNS lookup"],
  "/history": ["Monitoring", "History"],
  "/alerts": ["Monitoring", "Alert rules"],
  "/workspace/team": ["Workspace", "Team"],
  "/workspace/billing": ["Workspace", "Billing"],
  "/settings": ["Workspace", "Settings"],
  "/profile": ["Account", "Profile"],
  "/security": ["Account", "Security"],
  "/admin": ["Admin", "Overview"],
  "/admin/users": ["Admin", "Users"],
  "/admin/workspaces": ["Admin", "Workspaces"],
  "/admin/system": ["Admin", "System"],
};

export function Topbar({ session }: { session: SessionResponse }) {
  const pathname = usePathname();
  const { toggleSidebar } = useShellStore();
  const [section, page] = titles[pathname] ?? ["", "Console"];
  const isDemo = session.mode === "demo";

  return (
    <header className="sticky top-0 z-20 flex h-14 items-center gap-3 border-b border-[var(--lp-border)] bg-[#141417]/85 px-4 backdrop-blur-md lg:px-8">
      <button
        type="button"
        onClick={toggleSidebar}
        aria-label="Open navigation"
        className="-ml-1 rounded-md p-1.5 text-[var(--lp-muted)] transition-colors hover:bg-white/[0.05] hover:text-[var(--lp-text)] lg:hidden"
      >
        <Menu className="h-5 w-5" />
      </button>

      <nav aria-label="Breadcrumb" className="flex min-w-0 items-center gap-2 text-[13px]">
        {section ? (
          <>
            <span className="hidden text-[var(--lp-subtle)] sm:inline">{section}</span>
            <span className="hidden text-[var(--lp-subtle)] sm:inline">/</span>
          </>
        ) : null}
        <span className="truncate font-medium text-[var(--lp-text)]">{page}</span>
      </nav>

      <div className="ml-auto flex items-center gap-2">
        {isDemo ? (
          <span className="hidden rounded-md border border-[rgba(251,191,36,0.25)] px-2 py-1 text-[12px] text-[var(--lp-amber)] sm:inline">Demo</span>
        ) : null}
        {pathname !== "/tcp-ping" ? (
          <Link href="/tcp-ping" className="console-btn console-btn-primary">
            <Plus className="h-3.5 w-3.5" /> New probe
          </Link>
        ) : null}
      </div>
    </header>
  );
}
