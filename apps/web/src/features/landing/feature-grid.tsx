import { Activity, BellRing, Globe, History, KeyRound, Radar, Radio, Users, type AppIcon } from "@/components/icons";

// Only what ships today; the roadmap lives elsewhere on the page.
const features: { icon: AppIcon; title: string; body: string }[] = [
  { icon: Activity, title: "Live ping", body: "ICMP and TCP ping with latency, loss, and jitter as replies arrive." },
  { icon: Radar, title: "Port scan", body: "Thousands of ports per run, with service names and handshake times." },
  { icon: Globe, title: "DNS lookup", body: "Every record type with TTLs, DNSSEC status, and mail checks." },
  { icon: Radio, title: "Uptime monitors", body: "HTTP(S), TCP, and ping checks that keep running with the tab closed." },
  { icon: BellRing, title: "Alerts anywhere", body: "Email, Slack, Discord, Telegram, or a signed webhook when a check fails." },
  { icon: History, title: "History and export", body: "Every probe is kept. Chart a target over time and export CSV or JSON." },
  { icon: Users, title: "Teams and roles", body: "Owners, admins, members, and read-only viewers, with two-factor on demand." },
  { icon: KeyRound, title: "API keys", body: "The console runs on a public API. Script the same calls with your own key." },
];

export function FeatureGrid() {
  return (
    <div>
      <div className="lp-reveal flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
        <h2 className="max-w-xl text-3xl font-semibold leading-[1.12] tracking-[-0.03em] sm:text-[44px]">
          <span className="text-[var(--lp-text)]">Everything you reach for</span>{" "}
          <span className="text-[var(--lp-muted)]">when the network acts up</span>
        </h2>
        <a
          href="#products"
          className="lp-btn h-11 shrink-0 self-start bg-[var(--lp-text)] px-5 text-md font-medium text-[#0b0b0d] hover:bg-white sm:self-auto"
        >
          See the console <span aria-hidden="true">→</span>
        </a>
      </div>

      <ul className="mt-14 grid grid-cols-1 gap-x-10 gap-y-12 sm:grid-cols-2 lg:grid-cols-4 lg:gap-y-14">
        {features.map((f) => (
          <li key={f.title} className="lp-reveal">
            <span className="grid h-12 w-12 place-items-center rounded-full bg-[var(--lp-bg-raised)]">
              <f.icon className="h-5 w-5 text-[var(--lp-accent)]" aria-hidden="true" />
            </span>
            <h3 className="mt-6 text-lg font-semibold tracking-[-0.01em] text-[var(--lp-text)]">{f.title}</h3>
            <p className="mt-2.5 max-w-[17rem] text-base leading-relaxed text-[var(--lp-muted)]">{f.body}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}
