"use client";

import Link from "next/link";
import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { usePathname } from "next/navigation";
import { useShellStore } from "@/store/ui-store";
import { SessionResponse } from "@/lib/types";

interface NavItem {
  href: string;
  label: string;
  icon: string;
}

interface NavGroup {
  label: string;
  icon: string;
  items?: NavItem[];
  defaultOpen?: boolean;
}

const navGroups: NavGroup[] = [
  {
    label: "Operations",
    icon: "OP",
    defaultOpen: true,
    items: [
      { href: "/", label: "Overview", icon: "OV" },
      { href: "/icmp-ping", label: "ICMP Ping", icon: "IC" },
      { href: "/tcp-ping", label: "TCP Ping", icon: "TC" },
      { href: "/port-scan", label: "Port Scan", icon: "PS" },
      { href: "/dns-lookup", label: "DNS Lookup", icon: "DN" },
      { href: "/history", label: "History", icon: "HI" },
    ],
  },
  {
    label: "Workspace",
    icon: "WS",
    defaultOpen: true,
    items: [
      { href: "/workspace/team", label: "Team", icon: "TM" },
      { href: "/workspace/billing", label: "Billing", icon: "BL" },
      { href: "/alerts", label: "Alert Rules", icon: "AL" },
      { href: "/settings", label: "Settings", icon: "ST" },
    ],
  },
  {
    label: "Account",
    icon: "AC",
    defaultOpen: true,
    items: [
      { href: "/profile", label: "Profile", icon: "PR" },
      { href: "/security", label: "Security", icon: "SC" },
    ],
  },
  {
    label: "Admin",
    icon: "AD",
    items: [
      { href: "/admin", label: "Overview", icon: "OV" },
      { href: "/admin/users", label: "Users", icon: "US" },
      { href: "/admin/workspaces", label: "Workspaces", icon: "WK" },
      { href: "/admin/system", label: "System", icon: "SY" },
    ],
  },
];

export function Sidebar({ session }: { session: SessionResponse }) {
  const pathname = usePathname();
  const { sidebarOpen, setSidebarOpen } = useShellStore();
  const isDemo = session.mode === "demo";
  const [openGroups, setOpenGroups] = useState<Set<string>>(() => {
    const initial = new Set<string>();
    navGroups.forEach((group) => {
      if (group.defaultOpen || group.items?.some((item) => item.href === pathname)) {
        initial.add(group.label);
      }
    });
    return initial;
  });

  const toggleGroup = (label: string) => {
    setOpenGroups((prev) => {
      const next = new Set(prev);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      return next;
    });
  };

  return (
    <>
      {sidebarOpen && (
        <div className="fixed inset-0 z-20 bg-background/60 backdrop-blur-sm lg:hidden" onClick={() => setSidebarOpen(false)} />
      )}

      <aside
        className={`fixed top-0 left-0 z-30 h-full w-64 transition-transform duration-300 lg:sticky lg:translate-x-0 ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="sidebar-shell flex h-full flex-col overflow-y-auto border-r border-sidebar-border">
          <div className="px-4 py-4">
            <span className="text-[10px] font-semibold uppercase tracking-[0.16em] text-cyan-100/40">Main</span>
          </div>

          <nav className="flex-1 space-y-0.5 px-2 pb-4">
            {isDemo ? (
              <div className="mx-2 mb-3 rounded-2xl border border-cyan-300/18 bg-cyan-300/10 px-3 py-3 text-xs leading-6 text-cyan-100">
                Shared demo access is active.
              </div>
            ) : null}

            {navGroups
              .filter((group) => group.label !== "Admin" || session.platform_role === "admin")
              .map((group) => {
                const isOpen = openGroups.has(group.label);
                const hasActive = group.items?.some((item) => item.href === pathname) ?? false;
                const hasChildren = (group.items?.length ?? 0) > 0;

                return (
                  <div key={group.label}>
                    <button
                      type="button"
                      onClick={() => hasChildren && toggleGroup(group.label)}
                      className={`flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
                        hasActive ? "text-foreground" : "text-muted-foreground hover:bg-cyan-300/6 hover:text-foreground"
                      }`}
                    >
                      <span className="flex items-center gap-2.5">
                        <span className={`grid h-7 w-7 place-items-center rounded-md text-[10px] font-bold tracking-wider ${hasActive ? "bg-[linear-gradient(135deg,rgba(23,211,194,0.94),rgba(52,129,255,0.92))] text-slate-950 shadow-[0_0_14px_rgba(64,240,255,0.14)]" : "bg-secondary/60"}`}>
                          {group.icon}
                        </span>
                        <span className="font-medium">{group.label}</span>
                      </span>
                      <ChevronDown className={`h-3.5 w-3.5 transition-transform duration-200 ${isOpen ? "rotate-0" : "-rotate-90"}`} />
                    </button>

                    {hasChildren && isOpen ? (
                      <div className="mt-0.5 mb-1 ml-3 space-y-0.5 border-l border-border/40 pl-4">
                        {group.items?.map((item) => (
                          <Link
                            key={item.href}
                            href={item.href}
                            onClick={() => setSidebarOpen(false)}
                            className={`flex w-full items-center gap-2.5 rounded-md px-3 py-1.5 text-sm transition-all ${
                              pathname === item.href
                                ? "bg-cyan-300/12 font-medium text-cyan-100 shadow-[0_0_14px_rgba(64,240,255,0.06)]"
                                : "text-muted-foreground hover:bg-cyan-300/8 hover:text-foreground"
                            }`}
                          >
                            <span className={`grid h-6 w-6 place-items-center rounded text-[9px] font-bold tracking-wider ${pathname === item.href ? "bg-cyan-300/14 text-cyan-100" : "bg-secondary/40 text-[9px]"}`}>
                              {item.icon}
                            </span>
                            {item.label}
                          </Link>
                        ))}
                      </div>
                    ) : null}
                  </div>
                );
              })}
          </nav>
        </div>
      </aside>
    </>
  );
}
