import { QnetwardMark } from "./qnetward-mark";

// Static latency series (ms) for the illustrative chart.
const latency = [22, 24, 21, 26, 23, 25, 31, 28, 24, 22, 23, 27, 46, 38, 26, 23, 22, 24, 23, 25, 22, 21, 24, 23, 26, 24, 22, 23];

const targets = [
  { name: "api.acme.dev", proto: "HTTPS", latency: "23 ms", loss: "0%", state: "up" },
  { name: "10.0.4.1 · core-sw-01", proto: "ICMP", latency: "1.4 ms", loss: "0%", state: "up" },
  { name: "db.internal:5432", proto: "TCP", latency: "3.1 ms", loss: "0%", state: "up" },
  { name: "edge-sgp-02", proto: "ICMP", latency: "184 ms", loss: "12%", state: "warn" },
  { name: "mail.acme.dev", proto: "DNS", latency: "—", loss: "—", state: "down" },
];

const stateColor: Record<string, string> = {
  up: "var(--lp-green)",
  warn: "var(--lp-amber)",
  down: "var(--lp-accent)",
};

function chartPath(values: number[], width: number, height: number) {
  const max = 50;
  const step = width / (values.length - 1);
  return values.map((v, i) => `${i === 0 ? "M" : "L"}${(i * step).toFixed(1)},${(height - (v / max) * height).toFixed(1)}`).join(" ");
}

export function HeroConsole() {
  const line = chartPath(latency, 560, 120);

  return (
    <div className="relative mx-auto max-w-[1040px]" aria-hidden="true">
      <div
        className="pointer-events-none absolute -inset-x-10 -top-10 bottom-0 rounded-[40px] opacity-70 blur-3xl"
        style={{ background: "radial-gradient(ellipse at 50% 0%, rgba(255,61,127,0.25), transparent 65%)" }}
      />

      <div className="relative overflow-hidden rounded-2xl border border-[var(--lp-border-strong)] bg-[#17171b] shadow-[0_40px_120px_rgba(0,0,0,0.55)]">
        {/* window chrome */}
        <div className="flex h-10 items-center gap-2 border-b border-[var(--lp-border)] px-4">
          <span className="h-2.5 w-2.5 rounded-full bg-white/15" />
          <span className="h-2.5 w-2.5 rounded-full bg-white/15" />
          <span className="h-2.5 w-2.5 rounded-full bg-white/15" />
          <span className="lp-mono ml-4 text-[11px] text-[var(--lp-subtle)]">console.qnetward.io/overview</span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-[180px_1fr]">
          {/* sidebar */}
          <aside className="hidden border-r border-[var(--lp-border)] p-4 md:block">
            <div className="mb-5 flex items-center gap-2">
              <QnetwardMark size={20} />
              <span className="text-[13px] font-semibold">Qnetward</span>
            </div>
            {["Overview", "ICMP Ping", "TCP Ping", "Port Scan", "DNS Lookup", "History", "Alert Rules"].map((item, i) => (
              <div
                key={item}
                className="mb-1 rounded-md px-2.5 py-1.5 text-[12px]"
                style={{
                  background: i === 0 ? "rgba(255,255,255,0.06)" : undefined,
                  color: i === 0 ? "var(--lp-text)" : "var(--lp-subtle)",
                }}
              >
                {item}
              </div>
            ))}
          </aside>

          {/* main */}
          <div className="p-5">
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {[
                ["Targets", "24"],
                ["Healthy", "21"],
                ["Avg latency", "23 ms"],
                ["Open alerts", "2"],
              ].map(([label, value]) => (
                <div key={label} className="rounded-lg border border-[var(--lp-border)] bg-white/[0.02] p-3">
                  <div className="text-[11px] text-[var(--lp-subtle)]">{label}</div>
                  <div className="mt-1 text-lg font-semibold">{value}</div>
                </div>
              ))}
            </div>

            <div className="mt-4 rounded-lg border border-[var(--lp-border)] bg-white/[0.02] p-4">
              <div className="mb-3 flex items-center justify-between">
                <span className="text-[12px] font-medium">Latency · api.acme.dev</span>
                <span className="flex items-center gap-1.5 text-[11px] text-[var(--lp-green)]">
                  <span className="lp-pulse h-1.5 w-1.5 rounded-full bg-[var(--lp-green)]" />
                  Live
                </span>
              </div>
              <svg viewBox="0 0 560 120" className="h-[110px] w-full" preserveAspectRatio="none">
                <defs>
                  <linearGradient id="lp-area" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#ff3d7f" stopOpacity="0.35" />
                    <stop offset="100%" stopColor="#ff3d7f" stopOpacity="0" />
                  </linearGradient>
                </defs>
                {[30, 60, 90].map((y) => (
                  <line key={y} x1="0" x2="560" y1={y} y2={y} stroke="rgba(255,255,255,0.05)" />
                ))}
                <path d={`${line} L560,120 L0,120 Z`} fill="url(#lp-area)" />
                <path d={line} fill="none" stroke="#ff3d7f" strokeWidth="2" />
              </svg>
            </div>

            <div className="mt-4 overflow-hidden rounded-lg border border-[var(--lp-border)]">
              {targets.map((t, i) => (
                <div
                  key={t.name}
                  className="grid grid-cols-[1fr_auto_auto] items-center gap-4 px-3 py-2 text-[12px] sm:grid-cols-[1fr_64px_64px_48px]"
                  style={{ borderTop: i === 0 ? undefined : "1px solid var(--lp-border)" }}
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ background: stateColor[t.state] }} />
                    <span className="truncate">{t.name}</span>
                  </span>
                  <span className="lp-mono text-[11px] text-[var(--lp-subtle)]">{t.proto}</span>
                  <span className="lp-mono text-right">{t.latency}</span>
                  <span className="lp-mono hidden text-right text-[var(--lp-subtle)] sm:block">{t.loss}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* floating workspace switcher */}
      <div className="absolute -left-4 top-24 hidden w-56 rounded-xl border border-[var(--lp-border-strong)] bg-[#1d1d22]/95 p-2 shadow-2xl backdrop-blur lg:block">
        <div className="px-2 pb-2 pt-1 text-[10px] uppercase tracking-wider text-[var(--lp-subtle)]">Workspaces</div>
        {[
          ["Acme Production", "Owner", true],
          ["Branch Offices", "Admin", false],
          ["Client · NetCo ISP", "Member", false],
        ].map(([name, role, active]) => (
          <div
            key={name as string}
            className="flex items-center justify-between rounded-lg px-2 py-2 text-[12px]"
            style={{ background: active ? "rgba(255,61,127,0.12)" : undefined }}
          >
            <span>{name}</span>
            <span className="text-[10px] text-[var(--lp-subtle)]">{role}</span>
          </div>
        ))}
      </div>

      {/* floating alert toast */}
      <div className="absolute -right-4 bottom-16 hidden w-64 rounded-xl border border-[var(--lp-border-strong)] bg-[#1d1d22]/95 p-3 shadow-2xl backdrop-blur lg:block">
        <div className="flex items-center gap-2 text-[12px] font-medium">
          <span className="h-2 w-2 rounded-full bg-[var(--lp-amber)]" />
          Alert · packet loss above 10%
        </div>
        <div className="mt-1 text-[11px] text-[var(--lp-subtle)]">edge-sgp-02 · 12% loss over 5 probes · emailed on-call</div>
      </div>
    </div>
  );
}
