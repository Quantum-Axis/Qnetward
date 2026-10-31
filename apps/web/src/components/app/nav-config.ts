import {
  Activity,
  Bell,
  Building2,
  CreditCard,
  Gauge,
  Globe,
  History,
  LayoutDashboard,
  Radar,
  Server,
  Settings,
  Shield,
  User,
  Users,
  Zap,
  type LucideIcon,
} from "lucide-react";

export type NavItem = { href: string; label: string; icon: LucideIcon; keywords?: string };
export type NavSection = { label: string; items: NavItem[]; adminOnly?: boolean };

/** Single source for the sidebar, the command palette, and page titles. */
export const navSections: NavSection[] = [
  {
    label: "Diagnostics",
    items: [
      { href: "/overview", label: "Overview", icon: LayoutDashboard, keywords: "home dashboard" },
      { href: "/icmp-ping", label: "ICMP ping", icon: Activity, keywords: "ping latency loss" },
      { href: "/tcp-ping", label: "TCP ping", icon: Zap, keywords: "port handshake connect" },
      { href: "/port-scan", label: "Port scan", icon: Radar, keywords: "ports open services" },
      { href: "/dns-lookup", label: "DNS lookup", icon: Globe, keywords: "records mx txt dnssec" },
    ],
  },
  {
    label: "Monitoring",
    items: [
      { href: "/history", label: "History", icon: History, keywords: "results export csv" },
      { href: "/alerts", label: "Alert rules", icon: Bell, keywords: "notifications thresholds email" },
    ],
  },
  {
    label: "Workspace",
    items: [
      { href: "/workspace/team", label: "Team", icon: Users, keywords: "members invite roles" },
      { href: "/workspace/billing", label: "Billing", icon: CreditCard, keywords: "plan subscription" },
      { href: "/settings", label: "Settings", icon: Settings, keywords: "deployment status runtime" },
    ],
  },
  {
    label: "Account",
    items: [
      { href: "/profile", label: "Profile", icon: User, keywords: "name email" },
      { href: "/security", label: "Security", icon: Shield, keywords: "password sessions" },
    ],
  },
  {
    label: "Admin",
    adminOnly: true,
    items: [
      { href: "/admin", label: "Admin overview", icon: Gauge },
      { href: "/admin/users", label: "Users", icon: User },
      { href: "/admin/workspaces", label: "Workspaces", icon: Building2 },
      { href: "/admin/system", label: "System", icon: Server },
    ],
  },
];

/** Sections shown in the sidebar (Account lives in the user menu instead). */
export const sidebarSections = (isAdmin: boolean) =>
  navSections.filter((s) => s.label !== "Account" && (!s.adminOnly || isAdmin));

export const paletteSections = (isAdmin: boolean) => navSections.filter((s) => !s.adminOnly || isAdmin);

export function findNav(pathname: string) {
  for (const section of navSections) {
    const item = section.items.find((i) => i.href === pathname);
    if (item) return { section: section.label, item };
  }
  return null;
}
