import Link from "next/link";
import { CodeTabs } from "./code-tabs";
import { Faq } from "./faq";
import { HeroConsole } from "./hero-console";
import { LandingEffects } from "./landing-effects";
import { GitHubIcon, LandingNav } from "./landing-nav";
import { NetworkCanvas } from "./network-canvas";
import { Pricing } from "./pricing";
import { ProductShowcase } from "./product-showcase";
import { RoadmapPreview } from "./roadmap-preview";
import { QnetwardMark } from "./qnetward-mark";
import { TechLogos } from "./tech-logos";
import { ToolComparison } from "./tool-comparison";
import { WorldHexMap } from "./world-hex-map";

const footer: { title: string; links: [string, string][] }[] = [
  {
    title: "Product",
    links: [
      ["Ping & TCP ping", "#products"],
      ["Port scan", "#products"],
      ["DNS lookup", "#products"],
      ["Alert rules", "#products"],
      ["Pricing", "#pricing"],
    ],
  },
  {
    title: "Platform",
    links: [
      ["Roadmap", "/roadmap"],
      ["Probe agents", "/roadmap#theme-reach"],
      ["Attack surface", "/roadmap#theme-secure"],
      ["Global network", "#network"],
    ],
  },
  {
    title: "Developers",
    links: [
      ["CLI", "#developers"],
      ["REST API", "#developers"],
      ["Live WebSocket", "#developers"],
      ["GitHub", "https://github.com/Quantum-Axis/Qnetward"],
    ],
  },
  {
    title: "Company",
    links: [
      ["Security", "#security"],
      ["FAQ", "#faq"],
      ["Sign in", "/login"],
    ],
  },
];

const steps = [
  {
    title: "Create a workspace",
    body: "Sign up with an email address. A workspace is created for you, and you can invite teammates now or later.",
    hint: "~30 seconds",
  },
  {
    title: "Point it at something",
    body: "Ping a host, scan its ports, or pull its DNS records. Results stream in as they happen and are kept in history.",
    hint: "tcpping db.internal:5432",
  },
  {
    title: "Decide what's worth waking up for",
    body: "Add a latency or loss threshold to any target. When it trips, the people on that workspace get an email.",
    hint: "loss > 10% for 5 probes",
  },
];


const securityToday = [
  "Passwords hashed with bcrypt; sessions and reset tokens stored only as hashes",
  "Rate limits on sign-in, verification, and probe endpoints",
  "Strict origin checks on the API and live WebSocket",
  "Workspace data visible only to its members, by role",
  "Probes to private and cloud-metadata addresses blocked by default",
];

const securityBeforeScans = [
  "Proof of ownership via DNS TXT record or HTTP token",
  "Per-target scope, rate limit, and a kill switch",
  "An audit log entry for every scan: who, what, when",
];

const scanTiers = [
  {
    tier: "Tier 0",
    name: "Passive",
    who: "Any domain",
    body: "Public signals only: certificates, DNS records, and HTTP headers. Nothing is sent that a browser wouldn't send.",
  },
  {
    tier: "Tier 1",
    name: "Active",
    who: "Verified owners",
    body: "Port and service discovery, unlocked only after you prove ownership with a DNS TXT record or an HTTP token.",
  },
  {
    tier: "Tier 2",
    name: "Deep",
    who: "By approval",
    body: "Authenticated or intrusive checks. Needs a paid plan, signed consent, and a manual review before the first run.",
  },
];

function SectionHeader({
  label,
  title,
  aside,
}: {
  label?: string;
  title: string;
  aside?: React.ReactNode;
}) {
  return (
    <div className="lp-reveal grid gap-6 lg:grid-cols-12 lg:items-end">
      <div className={aside ? "lg:col-span-7" : "lg:col-span-12"}>
        {label ? <div className="mb-4 text-[13px] text-[var(--lp-accent)]">{label}</div> : null}
        <h2 className="text-[32px] font-normal leading-[1.12] tracking-[-0.025em] sm:text-[44px]">{title}</h2>
      </div>
      {aside ? <div className="text-[16px] leading-relaxed text-[var(--lp-muted)] lg:col-span-5 lg:pb-1.5">{aside}</div> : null}
    </div>
  );
}

function Check({ muted = false }: { muted?: boolean }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true" className="mt-[3px] shrink-0">
      {muted ? (
        <circle cx="8" cy="8" r="5.5" stroke="var(--lp-subtle)" strokeDasharray="2 2" />
      ) : (
        <path d="M3.5 8.5 6.5 11.5 12.5 4.5" stroke="var(--lp-green)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
      )}
    </svg>
  );
}

export function LandingPage() {
  return (
    <>
      <LandingEffects />
      {/* announcement */}
      <div className="border-b border-[var(--lp-border)] bg-[var(--lp-bg-raised)]">
        <div className="lp-container flex h-10 items-center justify-center gap-3 text-[13px]">
          <span className="lp-badge lp-badge-new">New</span>
          <span className="text-[var(--lp-muted)]">
            <span className="hidden sm:inline">Team workspaces, invitations, and roles are live</span>
            <span className="sm:hidden">Team workspaces are live</span>
          </span>
          <Link href="/signup" className="text-[var(--lp-text)] underline-offset-4 hover:underline">
            Try it →
          </Link>
        </div>
      </div>

      <LandingNav />

      <main id="main">
        {/* hero */}
        <section className="relative overflow-hidden">
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <div className="lp-aurora lp-aurora-a" />
            <div className="lp-aurora lp-aurora-b" />
          </div>
          <NetworkCanvas className="absolute inset-x-0 top-0 h-[760px] w-full opacity-80 [mask-image:linear-gradient(to_bottom,black_55%,transparent)]" />
          <div className="lp-container relative pb-16 pt-14 text-center sm:pt-20">
            <div className="mx-auto inline-flex items-center gap-2 rounded-full border border-[var(--lp-border)] bg-white/[0.03] px-3 py-1 text-[12px] text-[var(--lp-muted)]">
              <span className="lp-pulse h-1.5 w-1.5 rounded-full bg-[var(--lp-green)]" />
              Network · Infrastructure · Security
            </div>
            <h1 className="mx-auto mt-5 max-w-4xl text-[44px] font-normal leading-[1.08] tracking-[-0.035em] sm:text-[72px]">
              <span className="lp-hero-text">See every network, server, and threat</span>
              <span className="lp-cursor" />
            </h1>
            <p className="mx-auto mt-5 max-w-3xl text-[18px] leading-relaxed text-[var(--lp-muted)]">
              Qnetward probes, monitors, and protects your infrastructure from one platform.{" "}
              <br className="hidden md:block" />
              Live diagnostics today, agents, device telemetry, and attack-surface scanning next.
            </p>
            <div className="mt-7 flex flex-wrap items-center justify-center gap-3">
              <Link href="/signup" className="lp-btn lp-btn-primary h-11 px-6 text-[15px]">
                Start monitoring free
              </Link>
              <a href="#how" className="lp-btn lp-btn-secondary h-11 px-6 text-[15px]">
                See how it works
              </a>
            </div>
            <ul className="mt-5 flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-[13px] text-[var(--lp-subtle)]">
              {["Free to start", "No credit card", "Self-host ready"].map((item) => (
                <li key={item} className="flex items-center gap-2">
                  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                    <path d="M3.5 8.5 6.5 11.5 12.5 4.5" stroke="var(--lp-green)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                  {item}
                </li>
              ))}
            </ul>
            <div className="lp-reveal mt-10 sm:mt-12">
              <HeroConsole />
            </div>
          </div>
        </section>

        {/* stack strip */}
        <section className="lp-divider py-14">
          <div className="lp-container lp-reveal">
            <TechLogos />
          </div>
        </section>

        {/* comparison */}
        <section className="lp-divider py-28">
          <div className="lp-container">
            <SectionHeader
              label="Why one tool"
              title="Five tabs during an outage is four too many"
              aside="Most teams stitch diagnostics together from terminal commands, a website or two, and a separate uptime service. Qnetward puts them behind one login, with shared history."
            />
            <div className="mt-14">
              <ToolComparison />
            </div>
          </div>
        </section>

        {/* how it works */}
        <section id="how" className="lp-divider scroll-mt-20 py-28">
          <div className="lp-container grid grid-cols-1 gap-14 lg:grid-cols-12">
            <div className="lg:col-span-5">
              <div className="lg:sticky lg:top-28">
                <SectionHeader label="How it works" title="Three steps, no install" />
                <p className="lp-reveal mt-6 max-w-sm text-[16px] leading-relaxed text-[var(--lp-muted)]">
                  The console runs every probe from our side, so you can start before you&apos;ve decided where an
                  agent should live.
                </p>
                <Link href="/signup" className="lp-reveal mt-8 inline-flex items-center gap-2 text-[14px] text-[var(--lp-text)] hover:text-[var(--lp-accent)]">
                  Create a workspace <span aria-hidden="true">→</span>
                </Link>
              </div>
            </div>
            <ol className="lg:col-span-7">
              {steps.map((step, i) => (
                <li key={step.title} className="lp-reveal grid grid-cols-[48px_1fr] gap-x-4 border-t border-[var(--lp-border)] py-9 last:border-b">
                  <span className="lp-mono pt-1 text-[13px] text-[var(--lp-subtle)]">{String(i + 1).padStart(2, "0")}</span>
                  <div>
                    <h3 className="text-[22px] font-normal tracking-[-0.01em]">{step.title}</h3>
                    <p className="mt-3 max-w-lg text-[15px] leading-relaxed text-[var(--lp-muted)]">{step.body}</p>
                    <span className="lp-mono mt-5 inline-block text-[12px] text-[var(--lp-subtle)]">{step.hint}</span>
                  </div>
                </li>
              ))}
            </ol>
          </div>
        </section>

        {/* products */}
        <section id="products" className="lp-divider scroll-mt-20 py-28">
          <div className="lp-container">
            <SectionHeader
              label="Product"
              title="The tools you open during an incident, in one tab"
              aside="Every screen here is the console as it ships today, captured from the demo workspace."
            />
            <div className="lp-reveal mt-14">
              <ProductShowcase />
            </div>

            <div className="mt-20">
              <RoadmapPreview />
            </div>
          </div>
        </section>

        {/* developers */}
        <section id="developers" className="lp-divider scroll-mt-20 py-28">
          <div className="lp-container grid grid-cols-1 items-start gap-14 lg:grid-cols-12">
            <div className="lg:col-span-5">
              <SectionHeader label="Developers" title="The console is a client like any other" />
              <p className="lp-reveal mt-6 text-[16px] leading-relaxed text-[var(--lp-muted)]">
                The same Go engine sits behind the CLI, the REST API, and the WebSocket stream the dashboard uses. If the
                console can do it, so can your script.
              </p>
              <dl className="lp-reveal mt-10 border-t border-[var(--lp-border)] text-[14px]">
                {[
                  ["Binary", "One static executable for Linux, macOS, and Windows"],
                  ["Output", "--json on every command, ready for jq or a pipeline"],
                  ["Control", "Start, stream, and cancel a run over one socket"],
                ].map(([k, v]) => (
                  <div key={k} className="grid grid-cols-[96px_1fr] gap-4 border-b border-[var(--lp-border)] py-4">
                    <dt className="text-[var(--lp-subtle)]">{k}</dt>
                    <dd className="text-[var(--lp-muted)]">{v}</dd>
                  </div>
                ))}
              </dl>
            </div>
            <div className="lp-reveal lg:col-span-7">
              <CodeTabs />
            </div>
          </div>
        </section>

        {/* security */}
        <section id="security" className="lp-divider scroll-mt-20 py-28">
          <div className="lp-container grid grid-cols-1 gap-14 lg:grid-cols-12">
            <div className="lg:col-span-5">
              <SectionHeader label="Security" title="A scanner should refuse to scan what isn't yours" />
              <p className="lp-reveal mt-6 text-[16px] leading-relaxed text-[var(--lp-muted)]">
                Deeper security scanning is on the way. The guardrails on the right ship first, and no active scan runs
                against a target until every one of them is in place.
              </p>
            </div>
            <div className="lp-reveal grid gap-px overflow-hidden rounded-2xl border border-[var(--lp-border)] bg-[var(--lp-border)] sm:grid-cols-2 lg:col-span-7">
              <div className="bg-[var(--lp-bg)] p-8">
                <h3 className="text-[13px] text-[var(--lp-muted)]">In place today</h3>
                <ul className="mt-6 space-y-4">
                  {securityToday.map((item) => (
                    <li key={item} className="flex gap-3 text-[14px] leading-relaxed text-[var(--lp-text)]">
                      <Check />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
              <div className="bg-[var(--lp-bg)] p-8">
                <h3 className="text-[13px] text-[var(--lp-muted)]">Required before any active scan</h3>
                <ul className="mt-6 space-y-4">
                  {securityBeforeScans.map((item) => (
                    <li key={item} className="flex gap-3 text-[14px] leading-relaxed text-[var(--lp-muted)]">
                      <Check muted />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
            <div className="lp-reveal lg:col-span-12">
              <div className="flex items-baseline justify-between gap-6 border-b border-[var(--lp-border)] pb-4">
                <h3 className="text-[15px] font-medium">How scanning will be gated</h3>
                <span className="text-[13px] text-[var(--lp-subtle)]">Planned · each tier unlocks the next</span>
              </div>
              <ol className="grid md:grid-cols-3">
                {scanTiers.map((t, i) => (
                  <li key={t.tier} className={`border-b border-[var(--lp-border)] py-6 md:border-b-0 md:pr-8 ${i > 0 ? "md:border-l md:pl-8" : ""}`}>
                    <div className="flex items-center gap-2 text-[13px]">
                      <span className="lp-mono text-[var(--lp-subtle)]">{t.tier}</span>
                      <span className="text-[var(--lp-subtle)]">·</span>
                      <span className="text-[var(--lp-muted)]">{t.who}</span>
                    </div>
                    <h4 className="mt-2 text-[17px] font-normal">{t.name}</h4>
                    <p className="mt-2 text-[14px] leading-relaxed text-[var(--lp-muted)]">{t.body}</p>
                  </li>
                ))}
              </ol>
            </div>
          </div>
        </section>

        {/* network */}
        <section id="network" className="lp-divider relative scroll-mt-20 overflow-hidden pt-28">
          <div className="lp-container relative z-10">
            <SectionHeader
              label="Network"
              title="One vantage point can't tell you whose problem it is"
              aside={
                <>
                  Probing from several regions separates a local blip from a regional outage. We start in Singapore and
                  add regions where customers need them.
                  <span className="mt-5 flex flex-wrap gap-x-5 gap-y-2 text-[13px] text-[var(--lp-subtle)]">
                    <span className="flex items-center gap-2">
                      <span className="h-2 w-2 rounded-full bg-[var(--lp-accent)]" /> Primary
                    </span>
                    <span className="flex items-center gap-2">
                      <span className="h-2 w-2 rounded-full bg-white" /> Planned
                    </span>
                    <span>Regions shown are planned locations</span>
                  </span>
                </>
              }
            />
          </div>
          {/* Glow sits on the full-width section so it can never be clipped by the map container. */}
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0"
            style={{ background: "radial-gradient(ellipse 28% 32% at 70% 58%, rgba(255,61,127,0.09), transparent 100%)" }}
          />
          <div className="relative mx-auto mt-10 max-w-[1400px] px-2 pb-20 sm:px-6">
            <div className="lp-reveal">
              <WorldHexMap />
            </div>
          </div>
        </section>

        {/* pricing */}
        <section id="pricing" className="lp-divider scroll-mt-20 py-28">
          <div className="lp-container">
            <SectionHeader
              label="Pricing"
              title="Free while we're in beta"
              aside="Use everything that exists today at no cost. Paid plans arrive with agents and multi-region probing, and we'll announce prices well before the beta ends."
            />
            <div className="mt-14">
              <Pricing />
            </div>
          </div>
        </section>

        {/* faq */}
        <section id="faq" className="lp-divider scroll-mt-20 py-28">
          <div className="lp-container grid grid-cols-1 gap-14 lg:grid-cols-12">
            <div className="lg:col-span-4">
              <SectionHeader label="FAQ" title="Common questions" />
              <p className="lp-reveal mt-6 text-[15px] leading-relaxed text-[var(--lp-muted)]">
                Not covered here? The console is free to explore, so the quickest answer is often to try it.
              </p>
            </div>
            <div className="lp-reveal lg:col-span-8">
              <Faq />
            </div>
          </div>
        </section>

        {/* closing CTA */}
        <section className="lp-divider relative overflow-hidden py-28">
          <div
            className="pointer-events-none absolute inset-0"
            style={{ background: "radial-gradient(ellipse 40% 70% at 15% 100%, rgba(255,61,127,0.14), transparent 70%)" }}
          />
          <div className="lp-container relative grid grid-cols-1 items-end gap-10 lg:grid-cols-12">
            <h2 className="lp-reveal text-[40px] font-normal leading-[1.08] tracking-[-0.03em] sm:text-[60px] lg:col-span-8">
              <span className="lp-hero-text">Start watching your network</span>
              <span className="lp-cursor" />
            </h2>
            <div className="lp-reveal lg:col-span-4">
              <p className="text-[16px] leading-relaxed text-[var(--lp-muted)]">
                Free to start, no card required. Your first probe runs in under a minute.
              </p>
              <div className="mt-6 flex flex-wrap gap-3">
                <Link href="/signup" className="lp-btn lp-btn-primary h-11 px-6 text-[15px]">
                  Create free account
                </Link>
                <Link href="/login" className="lp-btn lp-btn-secondary h-11 px-6 text-[15px]">
                  Sign in
                </Link>
              </div>
            </div>
          </div>
        </section>
      </main>

      {/* footer */}
      <footer className="lp-divider pb-10 pt-16">
        <div className="lp-container">
          <div className="grid gap-10 md:grid-cols-[1.4fr_repeat(4,1fr)]">
            <div>
              <Link href="/" className="flex items-center gap-2.5">
                <QnetwardMark size={24} />
                <span className="font-semibold">Qnetward</span>
              </Link>
              <p className="mt-4 max-w-xs text-sm leading-relaxed text-[var(--lp-muted)]">
                Unified network, infrastructure, and security observability.
              </p>
              <a
                href="https://github.com/Quantum-Axis/Qnetward"
                target="_blank"
                rel="noreferrer"
                className="mt-5 inline-flex items-center gap-2 text-sm text-[var(--lp-muted)] hover:text-[var(--lp-text)]"
              >
                <GitHubIcon /> GitHub
              </a>
            </div>
            {footer.map((col) => (
              <div key={col.title}>
                <h3 className="text-sm font-medium">{col.title}</h3>
                <ul className="mt-4 space-y-2.5">
                  {col.links.map(([label, href]) => (
                    <li key={label} className="text-sm text-[var(--lp-muted)]">
                      <a
                        href={href}
                        {...(href.startsWith("http") ? { target: "_blank", rel: "noreferrer" } : {})}
                        className="transition-colors hover:text-[var(--lp-text)]"
                      >
                        {label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
          <div className="mt-14 flex flex-col items-start justify-between gap-4 border-t border-[var(--lp-border)] pt-6 text-[13px] text-[var(--lp-subtle)] sm:flex-row sm:items-center">
            <span>© 2026 Qnetward. All rights reserved.</span>
            <div className="flex gap-5">
              <Link href="/terms" className="hover:text-[var(--lp-text)]">
                Terms
              </Link>
              <Link href="/privacy" className="hover:text-[var(--lp-text)]">
                Privacy
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </>
  );
}
