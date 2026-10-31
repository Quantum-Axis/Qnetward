import Link from "next/link";

type Plan = {
  name: string;
  price: string;
  note: string;
  blurb: string;
  features: string[];
  cta: { label: string; href: string };
  highlight?: boolean;
  available: boolean;
};

const plans: Plan[] = [
  {
    name: "Free",
    price: "$0",
    note: "during beta",
    blurb: "Everything that works today, for you and your team.",
    features: [
      "ICMP ping, TCP ping, port scan, DNS inspector",
      "Live streaming results and history",
      "Alert rules with email delivery",
      "Workspaces, invitations, and roles",
      "CSV and JSON export",
    ],
    cta: { label: "Start free", href: "/signup" },
    highlight: true,
    available: true,
  },
  {
    name: "Pro",
    price: "Coming soon",
    note: "",
    blurb: "For teams that need to watch continuously, from more places.",
    features: [
      "Probe agents for private networks",
      "Multi-region probing",
      "Longer history retention",
      "Slack, Telegram, Discord, and webhooks",
      "Device collector (SNMP, syslog)",
    ],
    cta: { label: "Start free, upgrade later", href: "/signup" },
    available: false,
  },
  {
    name: "Enterprise",
    price: "Custom",
    note: "",
    blurb: "For ISPs, MSPs, and regulated teams with specific requirements.",
    features: [
      "SSO (SAML, OIDC) and SCIM",
      "Dedicated or self-hosted deployment",
      "Authorized deep security scanning",
      "Audit log export and data residency",
      "Priority support and SLA",
    ],
    cta: { label: "Register interest", href: "/signup" },
    available: false,
  },
];

export function Pricing() {
  return (
    <div className="grid gap-4 lg:grid-cols-3">
      {plans.map((plan) => (
        <article
          key={plan.name}
          className="lp-spot lp-reveal flex flex-col rounded-2xl border p-8"
          style={{
            borderColor: plan.highlight ? "rgba(255,61,127,0.45)" : "var(--lp-border)",
            background: plan.highlight ? "linear-gradient(180deg, rgba(255,61,127,0.06), transparent 40%), var(--lp-bg-raised)" : "var(--lp-bg-raised)",
          }}
        >
          <div className="flex items-center justify-between">
            <h3 className="text-[15px] font-medium">{plan.name}</h3>
            {plan.available ? null : <span className="text-[12px] text-[var(--lp-subtle)]">Not yet available</span>}
          </div>
          <div className="mt-6 flex items-baseline gap-2">
            <span className="text-[34px] font-normal tracking-[-0.02em]">{plan.price}</span>
            {plan.note ? <span className="text-[13px] text-[var(--lp-muted)]">{plan.note}</span> : null}
          </div>
          <p className="mt-3 text-[14px] leading-relaxed text-[var(--lp-muted)]">{plan.blurb}</p>
          <ul className="mt-8 flex-1 space-y-3 border-t border-[var(--lp-border)] pt-6">
            {plan.features.map((feature) => (
              <li key={feature} className="flex gap-3 text-[14px] leading-relaxed" style={{ color: plan.available ? "var(--lp-text)" : "var(--lp-muted)" }}>
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true" className="mt-[3px] shrink-0">
                  <path
                    d="M3.5 8.5 6.5 11.5 12.5 4.5"
                    stroke={plan.available ? "var(--lp-green)" : "var(--lp-subtle)"}
                    strokeWidth="1.8"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
                {feature}
              </li>
            ))}
          </ul>
          <Link
            href={plan.cta.href}
            className={`lp-btn mt-8 w-full ${plan.highlight ? "lp-btn-primary" : "lp-btn-secondary"}`}
          >
            {plan.cta.label}
          </Link>
        </article>
      ))}
    </div>
  );
}
