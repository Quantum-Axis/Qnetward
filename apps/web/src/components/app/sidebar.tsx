"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Bell,
  Building2,
  Check,
  ChevronsUpDown,
  CreditCard,
  Gauge,
  Globe,
  History,
  LayoutDashboard,
  LogOut,
  Radar,
  Server,
  Settings,
  Shield,
  User,
  Users,
  Zap,
  type LucideIcon,
} from "lucide-react";
import { apiRequest } from "@/lib/api";
import { SessionResponse, WorkspaceListResponse } from "@/lib/types";
import { useShellStore } from "@/store/ui-store";
import { QnetwardMark } from "@/features/landing/qnetward-mark";
import { cn } from "@/lib/utils";

type NavItem = { href: string; label: string; icon: LucideIcon };
type NavSection = { label: string; items: NavItem[]; adminOnly?: boolean };

const sections: NavSection[] = [
  {
    label: "Diagnostics",
    items: [
      { href: "/overview", label: "Overview", icon: LayoutDashboard },
      { href: "/icmp-ping", label: "ICMP ping", icon: Activity },
      { href: "/tcp-ping", label: "TCP ping", icon: Zap },
      { href: "/port-scan", label: "Port scan", icon: Radar },
      { href: "/dns-lookup", label: "DNS lookup", icon: Globe },
    ],
  },
  {
    label: "Monitoring",
    items: [
      { href: "/history", label: "History", icon: History },
      { href: "/alerts", label: "Alert rules", icon: Bell },
    ],
  },
  {
    label: "Workspace",
    items: [
      { href: "/workspace/team", label: "Team", icon: Users },
      { href: "/workspace/billing", label: "Billing", icon: CreditCard },
      { href: "/settings", label: "Settings", icon: Settings },
    ],
  },
  {
    label: "Admin",
    adminOnly: true,
    items: [
      { href: "/admin", label: "Overview", icon: Gauge },
      { href: "/admin/users", label: "Users", icon: User },
      { href: "/admin/workspaces", label: "Workspaces", icon: Building2 },
      { href: "/admin/system", label: "System", icon: Server },
    ],
  },
];

function initials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return value.trim().slice(0, 2).toUpperCase() || "?";
}

/** Closes a popover on outside click or Escape. */
function useDismiss(open: boolean, close: () => void) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close();
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, close]);
  return ref;
}

function WorkspaceSwitcher({ session }: { session: SessionResponse }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const ref = useDismiss(open, () => setOpen(false));
  const isDemo = session.mode === "demo";

  const list = useQuery({
    queryKey: ["workspace-list"],
    queryFn: () => apiRequest<WorkspaceListResponse>("/api/workspaces"),
    enabled: session.authenticated,
  });

  const switchWorkspace = useMutation({
    mutationFn: (workspaceID: number) =>
      apiRequest<{ ok: boolean; message: string }>("/api/workspaces", {
        method: "POST",
        body: JSON.stringify({ workspace_id: workspaceID }),
      }),
    onSuccess: async () => {
      setOpen(false);
      await queryClient.invalidateQueries();
      router.refresh();
    },
  });

  const workspaces = list.data?.workspaces ?? [];
  const activeID = list.data?.active_workspace_id ?? session.active_workspace_id;
  const active = workspaces.find((w) => w.id === activeID) ?? workspaces[0];
  const name = active?.name ?? (isDemo ? "Demo workspace" : "Workspace");

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex w-full items-center gap-2.5 rounded-lg px-2 py-2 text-left transition-colors hover:bg-white/[0.04]"
      >
        <QnetwardMark size={26} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] font-medium text-[var(--lp-text)]">{name}</span>
          <span className="block truncate text-[11px] text-[var(--lp-subtle)]">
            {isDemo ? "Shared sandbox" : active ? `${active.plan} · ${active.role}` : "Loading…"}
          </span>
        </span>
        <ChevronsUpDown className="h-4 w-4 shrink-0 text-[var(--lp-subtle)]" />
      </button>

      {open ? (
        <div role="menu" className="console-popover absolute left-0 right-0 top-full z-50 mt-1 p-1">
          <p className="px-2 pb-1 pt-1.5 text-[11px] text-[var(--lp-subtle)]">Workspaces</p>
          {workspaces.map((w) => (
            <button
              key={w.id}
              type="button"
              role="menuitem"
              disabled={switchWorkspace.isPending}
              onClick={() => (w.id === activeID ? setOpen(false) : switchWorkspace.mutate(w.id))}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] text-[var(--lp-text)] hover:bg-white/[0.05]"
            >
              <span className="grid h-5 w-5 place-items-center rounded bg-white/[0.06] text-[10px]">{initials(w.name)}</span>
              <span className="min-w-0 flex-1 truncate">{w.name}</span>
              {w.id === activeID ? <Check className="h-3.5 w-3.5 text-[var(--lp-accent)]" /> : null}
            </button>
          ))}
          <div className="my-1 h-px bg-[var(--lp-border)]" />
          <Link
            href="/workspace/team"
            role="menuitem"
            onClick={() => setOpen(false)}
            className="flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] text-[var(--lp-muted)] hover:bg-white/[0.05] hover:text-[var(--lp-text)]"
          >
            <Users className="h-3.5 w-3.5" /> Manage team
          </Link>
        </div>
      ) : null}
    </div>
  );
}

function UserMenu({ session }: { session: SessionResponse }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const ref = useDismiss(open, () => setOpen(false));
  const isDemo = session.mode === "demo";
  const displayName = session.user?.name?.trim() || session.user?.email || "Account";

  const logout = useMutation({
    mutationFn: () => apiRequest<{ ok: boolean }>("/api/auth/logout", { method: "POST", body: JSON.stringify({}) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["session"] });
      router.replace("/login");
    },
  });

  return (
    <div ref={ref} className="relative">
      {open ? (
        <div role="menu" className="console-popover absolute bottom-full left-0 right-0 z-50 mb-1 p-1">
          {[
            { href: "/profile", label: "Profile", icon: User },
            { href: "/security", label: "Security", icon: Shield },
          ].map((item) => (
            <Link
              key={item.href}
              href={item.href}
              role="menuitem"
              onClick={() => setOpen(false)}
              className="flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] text-[var(--lp-text)] hover:bg-white/[0.05]"
            >
              <item.icon className="h-3.5 w-3.5 text-[var(--lp-subtle)]" /> {item.label}
            </Link>
          ))}
          <div className="my-1 h-px bg-[var(--lp-border)]" />
          <button
            type="button"
            role="menuitem"
            onClick={() => logout.mutate()}
            disabled={logout.isPending}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] text-[#ff8fb3] hover:bg-[rgba(255,61,127,0.08)]"
          >
            <LogOut className="h-3.5 w-3.5" /> {logout.isPending ? "Signing out…" : isDemo ? "Leave demo" : "Sign out"}
          </button>
        </div>
      ) : null}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex w-full items-center gap-2.5 rounded-lg px-2 py-2 text-left transition-colors hover:bg-white/[0.04]"
      >
        <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-[linear-gradient(135deg,#ff3d7f,#8b5cf6)] text-[11px] font-semibold text-white">
          {initials(displayName)}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] text-[var(--lp-text)]">{displayName}</span>
          <span className="block truncate text-[11px] text-[var(--lp-subtle)]">{session.user?.email}</span>
        </span>
        <ChevronsUpDown className="h-4 w-4 shrink-0 text-[var(--lp-subtle)]" />
      </button>
    </div>
  );
}

export function Sidebar({ session }: { session: SessionResponse }) {
  const pathname = usePathname();
  const { sidebarOpen, setSidebarOpen } = useShellStore();
  const isDemo = session.mode === "demo";

  return (
    <>
      {sidebarOpen ? (
        <div className="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm lg:hidden" onClick={() => setSidebarOpen(false)} aria-hidden="true" />
      ) : null}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-40 flex w-60 flex-col border-r border-[var(--lp-border)] bg-[#111114] transition-transform duration-200 lg:sticky lg:top-0 lg:h-screen lg:translate-x-0",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className="p-2">
          <WorkspaceSwitcher session={session} />
        </div>

        <nav className="flex-1 overflow-y-auto px-2 pb-4" aria-label="Main">
          {sections
            .filter((section) => !section.adminOnly || session.platform_role === "admin")
            .map((section) => (
              <div key={section.label} className="mt-4 first:mt-2">
                <p className="px-2 pb-1 text-[11px] font-medium text-[var(--lp-subtle)]">{section.label}</p>
                <ul className="space-y-px">
                  {section.items.map((item) => {
                    const active = pathname === item.href;
                    return (
                      <li key={item.href}>
                        <Link
                          href={item.href}
                          onClick={() => setSidebarOpen(false)}
                          aria-current={active ? "page" : undefined}
                          className={cn(
                            "group relative flex items-center gap-2.5 rounded-md px-2 py-1.5 text-[13px] transition-colors",
                            active
                              ? "bg-white/[0.06] text-[var(--lp-text)]"
                              : "text-[var(--lp-muted)] hover:bg-white/[0.03] hover:text-[var(--lp-text)]",
                          )}
                        >
                          {active ? <span className="absolute -left-2 top-1.5 bottom-1.5 w-0.5 rounded-full bg-[var(--lp-accent)]" /> : null}
                          <item.icon className={cn("h-4 w-4", active ? "text-[var(--lp-accent)]" : "text-[var(--lp-subtle)] group-hover:text-[var(--lp-muted)]")} />
                          {item.label}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            ))}
        </nav>

        <div className="border-t border-[var(--lp-border)] p-2">
          {isDemo ? (
            <p className="mb-2 rounded-md border border-[rgba(251,191,36,0.25)] bg-[rgba(251,191,36,0.06)] px-2.5 py-2 text-[12px] leading-snug text-[var(--lp-amber)]">
              Shared demo workspace. Other visitors can see what you run here.
            </p>
          ) : null}
          <UserMenu session={session} />
        </div>
      </aside>
    </>
  );
}
