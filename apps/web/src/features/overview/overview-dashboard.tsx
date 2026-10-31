"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  Activity,
  AlertTriangle,
  Bell,
  Clock,
  Download,
  Gauge,
  Plus,
  RefreshCw,
  Search,
  Target,
  WifiOff,
  type LucideIcon,
} from "lucide-react";
import type { HistoryPoint, RuntimeHealthResponse } from "@/lib/types";
import type { OverviewTargetSnapshot } from "@/features/overview/overview-helpers";
import { cn } from "@/lib/utils";

const SERIES_COLORS = ["#ff3d7f", "#22c3d6", "#a78bfa", "#4ade80"];

const stateMeta: Record<OverviewTargetSnapshot["healthState"], { label: string; color: string; pill: string }> = {
  critical: { label: "Down", color: "#ff3d7f", pill: "border-[rgba(255,61,127,0.3)] bg-[rgba(255,61,127,0.1)] text-[#ff8fb3]" },
  warning: { label: "Degraded", color: "#fbbf24", pill: "border-[rgba(251,191,36,0.3)] bg-[rgba(251,191,36,0.1)] text-[#fcd34d]" },
  stable: { label: "Healthy", color: "#4ade80", pill: "border-[rgba(74,222,128,0.25)] bg-[rgba(74,222,128,0.08)] text-[#86efac]" },
  idle: { label: "No data", color: "#6c6c75", pill: "border-[var(--lp-border)] bg-white/[0.03] text-[var(--lp-muted)]" },
  disabled: { label: "Paused", color: "#6c6c75", pill: "border-[var(--lp-border)] bg-white/[0.03] text-[var(--lp-muted)]" },
};

function formatMs(value?: number | null) {
  if (typeof value !== "number" || Number.isNaN(value)) return "—";
  return value >= 100 ? `${Math.round(value)}` : value.toFixed(1);
}

function relativeTime(ts?: number | null) {
  if (!ts) return "—";
  const seconds = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  return hours < 24 ? `${hours}h ago` : `${Math.round(hours / 24)}d ago`;
}

function clockTime(ts: number) {
  return new Date(ts).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

/** Current time, refreshed every 10 seconds, so "streaming now" counts stay pure during render. */
function useNow(intervalMs = 10_000) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(id);
  }, [intervalMs]);
  return now;
}

/* ---------------------------------------------------------------- header */

export function WelcomeHeader({
  name,
  targets,
  health,
  onRefresh,
  refreshing,
}: {
  name?: string | null;
  targets: OverviewTargetSnapshot[];
  health?: RuntimeHealthResponse;
  onRefresh: () => void;
  refreshing: boolean;
}) {
  const deps = health ? [health.database, health.redis, health.object_storage].filter((d) => d.enabled) : [];
  const degraded = health?.ok === false || deps.some((d) => !d.healthy);
  const serviceLabel = !health ? "Checking services" : degraded ? "Service degraded" : "Services healthy";
  const serviceColor = !health ? "#6c6c75" : degraded ? "#fbbf24" : "#4ade80";
  const first = (name ?? "").trim().split(/\s+/)[0];
  const withData = targets.filter((t) => t.healthState !== "idle" && t.healthState !== "disabled");
  const healthy = withData.filter((t) => t.healthState === "stable").length;
  const pct = withData.length ? Math.round((healthy / withData.length) * 1000) / 10 : null;

  return (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <h1 className="text-[24px] font-medium tracking-[-0.02em]">
          Welcome back{first ? ", " : ""}
          {first ? <span className="text-[var(--lp-accent)]">{first}</span> : null}
        </h1>
        <p className="mt-1 text-[14px] text-[var(--lp-muted)]">
          {pct === null ? (
            "No probe results yet. Run one to start filling this page."
          ) : (
            <>
              <span className="text-[var(--lp-accent)]">{pct}%</span> of {withData.length} {withData.length === 1 ? "target is" : "targets are"} healthy right now.
            </>
          )}
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Link
          href="/settings"
          className="mr-1 hidden items-center gap-2 text-[12px] text-[var(--lp-muted)] hover:text-[var(--lp-text)] sm:flex"
          title="Database, cache, and object storage status"
        >
          <span className="h-2 w-2 rounded-full" style={{ background: serviceColor }} />
          {serviceLabel}
        </Link>
        <Link href="/tcp-ping" className="console-btn border border-[rgba(255,61,127,0.4)] bg-[rgba(255,61,127,0.08)] text-[#ff8fb3] hover:bg-[rgba(255,61,127,0.14)]">
          <Plus className="h-3.5 w-3.5" /> New probe
        </Link>
        <Link href="/alerts" className="console-btn console-btn-secondary">
          <AlertTriangle className="h-3.5 w-3.5" /> View alerts
        </Link>
        <button
          type="button"
          onClick={onRefresh}
          aria-label="Refresh"
          className="console-btn console-btn-secondary w-8 px-0"
          disabled={refreshing}
        >
          <RefreshCw className={cn("h-3.5 w-3.5", refreshing && "animate-spin")} />
        </button>
      </div>
    </div>
  );
}

/* ----------------------------------------------------------------- stats */

type Stat = { icon: LucideIcon; label: string; value: string; unit?: string; footer: string; tone?: "good" | "warn" | "bad" };

function StatCard({ stat }: { stat: Stat }) {
  const toneClass =
    stat.tone === "good"
      ? "border-[rgba(74,222,128,0.25)] bg-[rgba(74,222,128,0.08)] text-[#86efac]"
      : stat.tone === "warn"
        ? "border-[rgba(251,191,36,0.3)] bg-[rgba(251,191,36,0.08)] text-[#fcd34d]"
        : stat.tone === "bad"
          ? "border-[rgba(255,61,127,0.3)] bg-[rgba(255,61,127,0.08)] text-[#ff8fb3]"
          : "";
  return (
    <div className="flex flex-col rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f]">
      <div className="flex items-center gap-2.5 px-4 pt-4">
        <span className="grid h-7 w-7 place-items-center rounded-md border border-[var(--lp-border-strong)] bg-white/[0.03]">
          <stat.icon className="h-3.5 w-3.5 text-[var(--lp-muted)]" />
        </span>
        <span className="text-[13px] text-[var(--lp-text)]">{stat.label}</span>
      </div>
      <div className="flex items-baseline gap-1.5 px-4 pb-4 pt-3">
        <span className="text-[28px] font-medium tracking-[-0.02em]">{stat.value}</span>
        {stat.unit ? <span className="text-[13px] text-[var(--lp-subtle)]">{stat.unit}</span> : null}
      </div>
      <div className="mt-auto flex items-center justify-between gap-2 border-t border-[var(--lp-border)] px-4 py-2.5 text-[12px] text-[var(--lp-subtle)]">
        <span className="truncate">{stat.footer}</span>
        {stat.tone ? (
          <span className={cn("rounded border px-1.5 py-px text-[11px]", toneClass)}>
            {stat.tone === "good" ? "OK" : stat.tone === "warn" ? "Watch" : "High"}
          </span>
        ) : null}
      </div>
    </div>
  );
}

export function StatCards({
  targets,
  historyMap,
  alertRuleCount,
  firingCount,
}: {
  targets: OverviewTargetSnapshot[];
  historyMap: Map<string, HistoryPoint[]>;
  alertRuleCount: number;
  firingCount: number;
}) {
  const now = useNow();
  const stats = useMemo<Stat[]>(() => {
    const points = Array.from(historyMap.values()).flat();
    const rtts = points.map((p) => p.rtt_ms).filter((v): v is number => typeof v === "number");
    const avg = rtts.length ? rtts.reduce((a, b) => a + b, 0) / rtts.length : null;
    const failures = points.filter((p) => p.error || typeof p.rtt_ms !== "number").length;
    const loss = points.length ? (failures / points.length) * 100 : null;
    const live = targets.filter((t) => t.lastSampleAt && now - t.lastSampleAt < 15_000).length;
    return [
      {
        icon: Target,
        label: "Targets",
        value: String(targets.length),
        unit: targets.length === 1 ? "target" : "targets",
        footer: live ? `${live} streaming now` : "From recent probes and alert rules",
      },
      {
        icon: Gauge,
        label: "Average latency",
        value: formatMs(avg),
        unit: avg === null ? undefined : "ms",
        footer: rtts.length ? `Across ${rtts.length} recent replies` : "No replies yet",
        tone: avg === null ? undefined : avg < 100 ? "good" : avg < 250 ? "warn" : "bad",
      },
      {
        icon: WifiOff,
        label: "Packet loss",
        value: loss === null ? "—" : loss.toFixed(loss < 10 ? 1 : 0),
        unit: loss === null ? undefined : "%",
        footer: points.length ? `${failures} of ${points.length} probes failed` : "No probes yet",
        tone: loss === null ? undefined : loss < 1 ? "good" : loss < 5 ? "warn" : "bad",
      },
      {
        icon: Bell,
        label: "Alert rules",
        value: String(alertRuleCount),
        unit: alertRuleCount === 1 ? "rule" : "rules",
        footer: firingCount ? `${firingCount} currently firing` : alertRuleCount ? "None firing" : "No rules yet",
        tone: alertRuleCount ? (firingCount ? "bad" : "good") : undefined,
      },
    ];
  }, [alertRuleCount, firingCount, historyMap, now, targets]);

  return (
    <div className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
      {stats.map((stat) => (
        <StatCard key={stat.label} stat={stat} />
      ))}
    </div>
  );
}

/* ----------------------------------------------------------- latency chart */

type Series = { id: string; label: string; color: string; points: { ts: number; rtt: number }[] };

export function ResponseTimeChart({ targets, historyMap }: { targets: OverviewTargetSnapshot[]; historyMap: Map<string, HistoryPoint[]> }) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoverTs, setHoverTs] = useState<number | null>(null);

  const series = useMemo<Series[]>(() => {
    return targets
      .map((t) => ({
        id: t.id,
        label: t.name,
        points: (historyMap.get(t.id) ?? [])
          .filter((p): p is HistoryPoint & { rtt_ms: number } => typeof p.rtt_ms === "number")
          .map((p) => ({ ts: p.ts, rtt: p.rtt_ms }))
          .sort((a, b) => a.ts - b.ts),
      }))
      .filter((s) => s.points.length > 1)
      .slice(0, SERIES_COLORS.length)
      .map((s, i) => ({ ...s, color: SERIES_COLORS[i] }));
  }, [historyMap, targets]);

  const W = 800;
  const H = 240;
  const PAD = { l: 44, r: 12, t: 12, b: 26 };
  const all = series.flatMap((s) => s.points);
  const minTs = all.length ? Math.min(...all.map((p) => p.ts)) : 0;
  const maxTs = all.length ? Math.max(...all.map((p) => p.ts)) : 1;
  const maxRtt = all.length ? Math.max(...all.map((p) => p.rtt)) : 1;
  const minRtt = all.length ? Math.min(...all.map((p) => p.rtt)) : 0;
  const span = Math.max(1, maxRtt - minRtt);
  const yMin = Math.max(0, minRtt - span * 0.15);
  const yMax = maxRtt + span * 0.15;
  const x = (ts: number) => PAD.l + ((ts - minTs) / Math.max(1, maxTs - minTs)) * (W - PAD.l - PAD.r);
  const y = (rtt: number) => PAD.t + (1 - (rtt - yMin) / Math.max(1, yMax - yMin)) * (H - PAD.t - PAD.b);
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => yMin + (yMax - yMin) * f);

  const nearest = (s: Series, ts: number) => s.points.reduce((best, p) => (Math.abs(p.ts - ts) < Math.abs(best.ts - ts) ? p : best), s.points[0]);

  const onMove = (event: React.PointerEvent<SVGSVGElement>) => {
    const svg = svgRef.current;
    if (!svg || !all.length) return;
    const rect = svg.getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * W;
    const ts = minTs + ((px - PAD.l) / (W - PAD.l - PAD.r)) * (maxTs - minTs);
    setHoverTs(Math.min(maxTs, Math.max(minTs, ts)));
  };

  const tooltipLeft = hoverTs !== null ? (x(hoverTs) / W) * 100 : 0;
  // Only series that actually have data around the hovered time, so the tooltip never borrows a far-away value.
  const covering = hoverTs === null ? [] : series.filter((s) => hoverTs >= s.points[0].ts && hoverTs <= s.points[s.points.length - 1].ts);

  return (
    <section className="flex flex-col rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f]" aria-labelledby="rt-title">
      <div className="flex flex-wrap items-center gap-3 px-4 pt-4">
        <span className="grid h-7 w-7 place-items-center rounded-md border border-[var(--lp-border-strong)] bg-white/[0.03]">
          <Clock className="h-3.5 w-3.5 text-[var(--lp-muted)]" />
        </span>
        <h2 id="rt-title" className="text-[14px] font-medium">
          Response time
        </h2>
        <span className="ml-auto text-[12px] text-[var(--lp-subtle)]">Last 24 samples per target</span>
      </div>
      {series.length ? (
        <>
          <ul className="flex flex-wrap gap-x-4 gap-y-1 px-4 pt-3 text-[12px] text-[var(--lp-muted)]">
            {series.map((s) => (
              <li key={s.id} className="flex items-center gap-1.5">
                <span className="h-2.5 w-[3px] rounded-full" style={{ background: s.color }} />
                <span className="max-w-[180px] truncate">{s.label}</span>
              </li>
            ))}
          </ul>
          <div className="relative px-2 pb-3 pt-2">
            <svg
              ref={svgRef}
              viewBox={`0 0 ${W} ${H}`}
              className="h-[240px] w-full touch-none"
              preserveAspectRatio="none"
              onPointerMove={onMove}
              onPointerLeave={() => setHoverTs(null)}
              role="img"
              aria-label="Response time per target"
            >
              {ticks.map((v) => (
                <g key={v}>
                  <line x1={PAD.l} x2={W - PAD.r} y1={y(v)} y2={y(v)} stroke="rgba(255,255,255,0.05)" vectorEffect="non-scaling-stroke" />
                  <text x={PAD.l - 8} y={y(v) + 4} textAnchor="end" fontSize="11" fill="#6c6c75">
                    {Math.round(v)}ms
                  </text>
                </g>
              ))}
              {series.map((s) => (
                <path
                  key={s.id}
                  d={s.points.map((p, i) => `${i ? "L" : "M"}${x(p.ts).toFixed(1)},${y(p.rtt).toFixed(1)}`).join(" ")}
                  fill="none"
                  stroke={s.color}
                  strokeWidth="1.6"
                  strokeLinejoin="round"
                  vectorEffect="non-scaling-stroke"
                  opacity={hoverTs === null ? 1 : 0.9}
                />
              ))}
              {hoverTs !== null ? (
                <>
                  <line x1={x(hoverTs)} x2={x(hoverTs)} y1={PAD.t} y2={H - PAD.b} stroke="rgba(255,255,255,0.18)" strokeDasharray="3 3" vectorEffect="non-scaling-stroke" />
                  {covering.map((s) => {
                    const p = nearest(s, hoverTs);
                    return <circle key={s.id} cx={x(p.ts)} cy={y(p.rtt)} r="3.5" fill={s.color} stroke="#1b1b1f" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />;
                  })}
                </>
              ) : null}
            </svg>
            {hoverTs !== null && covering.length ? (
              <div
                className="console-popover pointer-events-none absolute top-6 z-10 w-56 p-3 text-[12px]"
                style={{ left: `clamp(8px, calc(${tooltipLeft}% + 12px), calc(100% - 232px))` }}
              >
                <div className="mb-2 text-[var(--lp-subtle)]">{clockTime(hoverTs)}</div>
                <ul className="space-y-1.5">
                  {covering.map((s) => (
                    <li key={s.id} className="flex items-center gap-2">
                      <span className="h-2.5 w-[3px] rounded-full" style={{ background: s.color }} />
                      <span className="min-w-0 flex-1 truncate text-[var(--lp-muted)]">{s.label}</span>
                      <span className="lp-mono text-[var(--lp-text)]">
                        {formatMs(nearest(s, hoverTs).rtt)} <span className="text-[var(--lp-subtle)]">ms</span>
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </div>
        </>
      ) : (
        <div className="m-4 flex flex-1 flex-col items-center justify-center rounded-lg border border-dashed border-[var(--lp-border)] py-16 text-center">
          <Activity className="h-5 w-5 text-[var(--lp-subtle)]" />
          <p className="mt-3 text-[13px] text-[var(--lp-muted)]">No latency data yet</p>
          <p className="mt-1 text-[12px] text-[var(--lp-subtle)]">Run a ping or TCP ping and each target gets its own line here.</p>
        </div>
      )}
    </section>
  );
}

/* ----------------------------------------------------------- status card */

function SegmentBar({ value, color }: { value: number; color: string }) {
  const segments = 28;
  const filled = Math.round((value / 100) * segments);
  return (
    <div className="flex gap-[3px]" aria-hidden="true">
      {Array.from({ length: segments }, (_, i) => (
        <span key={i} className="h-3.5 flex-1 rounded-[2px]" style={{ background: i < filled ? color : "rgba(255,255,255,0.07)" }} />
      ))}
    </div>
  );
}

export function TargetHealthCard({ targets }: { targets: OverviewTargetSnapshot[] }) {
  const total = targets.length;
  const rows: { state: OverviewTargetSnapshot["healthState"]; label: string }[] = [
    { state: "stable", label: "Healthy" },
    { state: "warning", label: "Degraded" },
    { state: "critical", label: "Down" },
    { state: "idle", label: "No recent data" },
  ];
  const attention = targets.filter((t) => t.healthState === "critical" || t.healthState === "warning");

  return (
    <section className="flex flex-col rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f]" aria-labelledby="health-title">
      <div className="flex items-center gap-3 px-4 pt-4">
        <span className="grid h-7 w-7 place-items-center rounded-md border border-[var(--lp-border-strong)] bg-white/[0.03]">
          <Activity className="h-3.5 w-3.5 text-[var(--lp-muted)]" />
        </span>
        <h2 id="health-title" className="text-[14px] font-medium">
          Target health
        </h2>
      </div>
      <div className="space-y-4 px-4 pt-4">
        {rows.map((row) => {
          const count = targets.filter((t) => t.healthState === row.state).length;
          const pct = total ? Math.round((count / total) * 100) : 0;
          return (
            <div key={row.state}>
              <div className="mb-1.5 flex items-baseline justify-between text-[13px]">
                <span className="text-[var(--lp-text)]">{row.label}</span>
                <span className="lp-mono text-[12px]" style={{ color: count ? stateMeta[row.state].color : "var(--lp-subtle)" }}>
                  {count} <span className="text-[var(--lp-subtle)]">· {pct}%</span>
                </span>
              </div>
              <SegmentBar value={pct} color={stateMeta[row.state].color} />
            </div>
          );
        })}
      </div>
      <div className="m-4 mt-5 rounded-lg border border-[var(--lp-border)] bg-black/20 p-3">
        {attention.length ? (
          <>
            <p className="flex items-center gap-2 text-[12px] text-[var(--lp-muted)]">
              <AlertTriangle className="h-3.5 w-3.5 text-[var(--lp-amber)]" />
              {attention.length} {attention.length === 1 ? "target needs" : "targets need"} attention
            </p>
            <ul className="mt-2 space-y-1.5 text-[12px]">
              {attention.slice(0, 3).map((t) => (
                <li key={t.id}>
                  <Link href={t.href} className="flex items-center gap-2 text-[var(--lp-text)] hover:text-[var(--lp-accent)]">
                    <span className="h-1.5 w-1.5 rounded-sm" style={{ background: stateMeta[t.healthState].color }} />
                    <span className="truncate">{t.name}</span>
                    <span className="ml-auto text-[var(--lp-subtle)]">{stateMeta[t.healthState].label}</span>
                  </Link>
                </li>
              ))}
            </ul>
          </>
        ) : (
          <p className="text-[12px] text-[var(--lp-subtle)]">{total ? "Nothing needs attention right now." : "Targets appear here after your first probe."}</p>
        )}
      </div>
    </section>
  );
}

/* ----------------------------------------------------------- targets table */

type StatusFilter = "all" | "stable" | "warning" | "critical" | "idle";
type TypeFilter = "all" | "ping" | "tcpping";

function SampleBars({ count }: { count: number }) {
  const level = count === 0 ? 0 : count < 8 ? 1 : count < 16 ? 2 : 3;
  return (
    <span className="inline-flex items-end gap-[2px]" aria-hidden="true">
      {[6, 9, 12].map((h, i) => (
        <span key={h} className="w-[3px] rounded-[1px]" style={{ height: h, background: i < level ? "#ff3d7f" : "rgba(255,255,255,0.12)" }} />
      ))}
    </span>
  );
}

export function TargetsSummary({ targets }: { targets: OverviewTargetSnapshot[] }) {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [type, setType] = useState<TypeFilter>("all");

  const rows = targets.filter(
    (t) =>
      (status === "all" || t.healthState === status) &&
      (type === "all" || t.protocol === type) &&
      (!query.trim() || t.name.toLowerCase().includes(query.trim().toLowerCase())),
  );

  return (
    <section className="overflow-hidden rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f]" aria-labelledby="summary-title">
      <div className="flex flex-wrap items-center gap-2 px-4 py-3">
        <h2 id="summary-title" className="mr-auto text-[14px] font-medium">
          Targets
        </h2>
        <label className="relative">
          <span className="sr-only">Search targets</span>
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[var(--lp-subtle)]" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search"
            className="h-8 w-40 pl-8 pr-2 text-[13px]"
          />
        </label>
        <select aria-label="Filter by status" value={status} onChange={(e) => setStatus(e.target.value as StatusFilter)} className="h-8 px-2 text-[13px]">
          <option value="all">All statuses</option>
          <option value="stable">Healthy</option>
          <option value="warning">Degraded</option>
          <option value="critical">Down</option>
          <option value="idle">No data</option>
        </select>
        <select aria-label="Filter by type" value={type} onChange={(e) => setType(e.target.value as TypeFilter)} className="h-8 px-2 text-[13px]">
          <option value="all">All types</option>
          <option value="ping">ICMP</option>
          <option value="tcpping">TCP</option>
        </select>
        <Link href="/history" aria-label="Export from history" className="console-btn console-btn-secondary h-8 w-8 px-0">
          <Download className="h-3.5 w-3.5" />
        </Link>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-[13px]">
          <thead>
            <tr className="border-y border-[var(--lp-border)] bg-white/[0.015] text-left text-[12px] text-[var(--lp-subtle)]">
              <th className="px-4 py-2.5 font-medium">Target</th>
              <th className="px-3 py-2.5 font-medium">Type</th>
              <th className="px-3 py-2.5 font-medium">Status</th>
              <th className="px-3 py-2.5 font-medium">Samples</th>
              <th className="px-3 py-2.5 text-right font-medium">Last RTT</th>
              <th className="px-4 py-2.5 text-right font-medium">Last seen</th>
            </tr>
          </thead>
          <tbody>
            {rows.length ? (
              rows.map((t) => (
                <tr key={t.id} className="border-b border-[var(--lp-border)] last:border-b-0 hover:bg-white/[0.02]">
                  <td className="px-4 py-3">
                    <Link href={t.href} className="text-[var(--lp-text)] hover:text-[var(--lp-accent)]">
                      {t.name}
                    </Link>
                  </td>
                  <td className="lp-mono px-3 py-3 text-[12px] text-[var(--lp-muted)]">{t.protocol === "tcpping" ? "TCP" : "ICMP"}</td>
                  <td className="px-3 py-3">
                    <span className={cn("inline-flex rounded-md border px-2 py-0.5 text-[12px]", stateMeta[t.healthState].pill)}>
                      {stateMeta[t.healthState].label}
                    </span>
                  </td>
                  <td className="px-3 py-3">
                    <span className="flex items-center gap-2">
                      <SampleBars count={t.pointCount} />
                      <span className="lp-mono text-[12px] text-[var(--lp-muted)]">{t.pointCount}</span>
                    </span>
                  </td>
                  <td className="lp-mono px-3 py-3 text-right">
                    {formatMs(t.lastRttMs)}
                    {typeof t.lastRttMs === "number" ? <span className="text-[var(--lp-subtle)]"> ms</span> : null}
                  </td>
                  <td className="px-4 py-3 text-right text-[var(--lp-subtle)]">{relativeTime(t.lastSampleAt)}</td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-4 py-10 text-center text-[13px] text-[var(--lp-subtle)]">
                  {targets.length ? "No targets match these filters." : "No targets yet. Run a probe and it will show up here."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------ coming next */

const upcoming = [
  { href: "/monitors", title: "Monitors", body: "Round-the-clock HTTP, TLS, and ping checks from several regions.", phase: 2 },
  { href: "/topology", title: "Topology", body: "A live map of your network built from LLDP and CDP.", phase: 3 },
  { href: "/attack-surface", title: "Attack surface", body: "Subdomains, open services, and certificates you didn't know about.", phase: 4 },
  { href: "/assistant", title: "AI assistant", body: "Ask about outages in plain language and get likely causes.", phase: 5 },
];

export function ComingNext() {
  return (
    <section aria-labelledby="next-title">
      <div className="mb-3 flex items-baseline justify-between">
        <h2 id="next-title" className="text-[14px] font-medium">
          Coming next
        </h2>
        <span className="text-[12px] text-[var(--lp-subtle)]">Previews with sample data</span>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {upcoming.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className="group rounded-xl border border-[var(--lp-border)] bg-[#1b1b1f] p-4 transition-colors hover:border-[var(--lp-border-strong)] hover:bg-[#202025]"
          >
            <div className="flex items-center justify-between">
              <span className="text-[13px] font-medium">{item.title}</span>
              <span className="rounded border border-[var(--lp-border)] px-1.5 text-[10px] leading-4 text-[var(--lp-subtle)]">Phase {item.phase}</span>
            </div>
            <p className="mt-2 text-[12px] leading-relaxed text-[var(--lp-muted)]">{item.body}</p>
            <span className="mt-3 inline-block text-[12px] text-[var(--lp-accent)] opacity-80 group-hover:opacity-100">See preview →</span>
          </Link>
        ))}
      </div>
    </section>
  );
}
