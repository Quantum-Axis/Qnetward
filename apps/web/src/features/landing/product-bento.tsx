// Product section: each tile shows a small, honest slice of the real UI instead of an icon.

const pingRows = [
  { seq: 14, ms: 18.4 },
  { seq: 15, ms: 18.1 },
  { seq: 16, ms: 41.7 },
  { seq: 17, ms: 18.9 },
  { seq: 18, ms: null },
  { seq: 19, ms: 18.2 },
];

const spark = [18, 19, 18, 20, 18, 19, 22, 18, 19, 41, 24, 19, 18, 18, 20, 19, 18, 19, 18, 21, 18, 19];

function Sparkline() {
  const w = 280;
  const h = 56;
  const max = 45;
  const step = w / (spark.length - 1);
  const d = spark.map((v, i) => `${i ? "L" : "M"}${(i * step).toFixed(1)},${(h - (v / max) * h).toFixed(1)}`).join(" ");
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="h-14 w-full" preserveAspectRatio="none" aria-hidden="true">
      <path d={`${d} L${w},${h} L0,${h} Z`} fill="rgba(255,61,127,0.10)" />
      <path d={d} fill="none" stroke="#ff3d7f" strokeWidth="1.5" />
    </svg>
  );
}

// 22 → open, 3 → filtered, others closed; laid out as a 12x4 port map.
const ports = Array.from({ length: 48 }, (_, i) => {
  if ([2, 7, 15, 30].includes(i)) return "open";
  if ([5, 21, 38].includes(i)) return "filtered";
  return "closed";
});

const portColor: Record<string, string> = {
  open: "#4ade80",
  filtered: "#fbbf24",
  closed: "rgba(255,255,255,0.07)",
};

const dnsRows = [
  ["A", "104.21.48.12"],
  ["AAAA", "2606:4700:3031::6815"],
  ["MX", "10 mx1.mailhost.net"],
  ["TXT", "v=spf1 include:_spf…"],
  ["CAA", '0 issue "letsencrypt.org"'],
];

const bars = [34, 52, 41, 66, 58, 72, 49, 61, 80, 57, 63, 70, 54, 68];

function Tile({
  title,
  body,
  className = "",
  horizontal = false,
  children,
}: {
  title: string;
  body: string;
  className?: string;
  horizontal?: boolean;
  children: React.ReactNode;
}) {
  return (
    <article
      className={`lp-spot lp-reveal flex flex-col overflow-hidden rounded-2xl border border-[var(--lp-border)] bg-[var(--lp-bg-raised)] ${horizontal ? "md:flex-row-reverse" : ""} ${className}`}
    >
      <div
        className={`relative flex-1 border-[var(--lp-border)] bg-[#17171b] p-5 ${horizontal ? "border-b md:border-b-0 md:border-l" : "border-b"}`}
      >
        {children}
      </div>
      <div className={horizontal ? "p-5 md:w-[34%] md:self-center md:p-8" : "p-5"}>
        <h3 className="text-[15px] font-medium">{title}</h3>
        <p className="mt-1.5 text-[14px] leading-relaxed text-[var(--lp-muted)]">{body}</p>
      </div>
    </article>
  );
}

export function ProductBento() {
  return (
    <div className="grid gap-4 md:grid-cols-6">
      <Tile
        className="md:col-span-4"
        title="Ping and TCP ping, streamed"
        body="Each reply lands the moment it arrives. Loss streaks, jitter, and percentiles update as you watch, and you can stop a run mid-flight."
      >
        <div className="flex items-center justify-between text-[12px]">
          <span className="lp-mono text-[var(--lp-muted)]">tcpping api.acme.dev:443</span>
          <span className="flex items-center gap-1.5 text-[var(--lp-green)]">
            <span className="lp-pulse h-1.5 w-1.5 rounded-full bg-[var(--lp-green)]" />
            running
          </span>
        </div>
        <div className="mt-4">
          <Sparkline />
        </div>
        <div className="lp-mono mt-4 grid grid-cols-2 gap-x-8 text-[12px] sm:grid-cols-3">
          {pingRows.map((r) => (
            <div key={r.seq} className="flex justify-between border-t border-[var(--lp-border)] py-1.5">
              <span className="text-[var(--lp-subtle)]">seq={r.seq}</span>
              {r.ms === null ? (
                <span className="text-[var(--lp-amber)]">timeout</span>
              ) : (
                <span style={{ color: r.ms > 30 ? "var(--lp-amber)" : "var(--lp-text)" }}>{r.ms} ms</span>
              )}
            </div>
          ))}
        </div>
      </Tile>

      <Tile
        className="md:col-span-2"
        title="Port scan"
        body="Up to 4,096 ports per run with 256 parallel connections."
      >
        <div className="grid grid-cols-12 gap-1">
          {ports.map((state, i) => (
            <span key={i} className="aspect-square rounded-[3px]" style={{ background: portColor[state] }} />
          ))}
        </div>
        <div className="lp-mono mt-5 space-y-1 text-[12px]">
          <div className="flex justify-between">
            <span>443/tcp</span>
            <span className="text-[var(--lp-green)]">open</span>
          </div>
          <div className="flex justify-between">
            <span>22/tcp</span>
            <span className="text-[var(--lp-amber)]">filtered</span>
          </div>
          <div className="flex justify-between text-[var(--lp-subtle)]">
            <span>3306/tcp</span>
            <span>closed</span>
          </div>
        </div>
      </Tile>

      <Tile className="md:col-span-2" title="DNS inspector" body="Fourteen record types, DNSSEC, and reverse lookups across public resolvers.">
        <table className="lp-mono w-full text-[12px]">
          <tbody>
            {dnsRows.map(([type, value]) => (
              <tr key={type} className="border-t border-[var(--lp-border)] first:border-t-0">
                <td className="w-14 py-1.5 text-[var(--lp-accent)]">{type}</td>
                <td className="truncate py-1.5 text-[var(--lp-muted)]">{value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Tile>

      <Tile className="md:col-span-2" title="Alert rules" body="Thresholds on latency and loss, with cooldowns so one bad minute is one email.">
        <div className="rounded-lg border border-[var(--lp-border)] bg-black/20 p-3 text-[12px]">
          <div className="lp-mono text-[var(--lp-muted)]">
            when <span className="text-[var(--lp-text)]">loss</span> &gt; <span className="text-[var(--lp-accent)]">10%</span>
          </div>
          <div className="lp-mono text-[var(--lp-muted)]">
            for <span className="text-[var(--lp-text)]">5 probes</span> on <span className="text-[var(--lp-text)]">edge-sgp-02</span>
          </div>
        </div>
        <div className="mt-3 flex items-start gap-2.5 rounded-lg border border-[var(--lp-border)] p-3 text-[12px]">
          <span className="mt-1 h-2 w-2 shrink-0 rounded-full bg-[var(--lp-amber)]" />
          <div>
            <div className="text-[var(--lp-text)]">Packet loss above 10%</div>
            <div className="text-[var(--lp-subtle)]">Emailed oncall@acme.dev · 2m ago</div>
          </div>
        </div>
      </Tile>

      <Tile className="md:col-span-2" title="Workspaces and roles" body="Invite your team, keep clients apart, and switch between workspaces in one click.">
        <ul className="space-y-2 text-[12px]">
          {[
            ["AR", "Ayesha Rahman", "Owner", "#ff3d7f"],
            ["TK", "Tanvir Khan", "Admin", "#8b5cf6"],
            ["MS", "Mitu Sultana", "Member", "#22c3d6"],
          ].map(([initials, name, role, color]) => (
            <li key={name} className="flex items-center gap-3">
              <span
                className="flex h-7 w-7 items-center justify-center rounded-full text-[10px] font-semibold text-white"
                style={{ background: color }}
              >
                {initials}
              </span>
              <span className="flex-1 text-[var(--lp-text)]">{name}</span>
              <span className="text-[var(--lp-subtle)]">{role}</span>
            </li>
          ))}
          <li className="flex items-center gap-3 text-[var(--lp-subtle)]">
            <span className="flex h-7 w-7 items-center justify-center rounded-full border border-dashed border-[var(--lp-border-strong)]">+</span>
            Invite pending · noc@client.io
          </li>
        </ul>
      </Tile>

      <Tile
        className="md:col-span-6"
        horizontal
        title="History you can query and export"
        body="Every probe is stored in TimescaleDB. Filter by target and protocol, chart it, and export a CSV or JSON for the incident review."
      >
        <div className="flex items-end gap-1.5" style={{ height: 96 }}>
          {bars.map((v, i) => (
            <span
              key={i}
              className="flex-1 rounded-t-[3px]"
              style={{ height: `${v}%`, background: i === 8 ? "#ff3d7f" : "rgba(255,255,255,0.12)" }}
            />
          ))}
        </div>
        <div className="mt-3 flex items-center justify-between text-[12px] text-[var(--lp-subtle)]">
          <span className="lp-mono">last 14 days · p95 latency</span>
          <span className="rounded-md border border-[var(--lp-border)] px-2 py-1 text-[var(--lp-text)]">Export CSV</span>
        </div>
      </Tile>
    </div>
  );
}
