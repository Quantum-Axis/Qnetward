const rows: { before: string; after: string }[] = [
  { before: "ping and mtr in a terminal", after: "Live ICMP and TCP ping with loss streaks and jitter" },
  { before: "nmap on someone's laptop", after: "Port scans saved to history, visible to the team" },
  { before: "dig, nslookup, and a DNS website", after: "DNS inspector with DNSSEC and reverse lookups" },
  { before: "A separate uptime checker", after: "HTTP(S), TCP, and ping monitors with certificate expiry warnings" },
  { before: "A spreadsheet of client sites", after: "One workspace per client, with roles" },
];

// A plain two-column list: what teams use today, and what replaces it here.
export function ToolComparison() {
  return (
    <div className="lp-reveal">
      <div className="hidden grid-cols-2 gap-8 border-b border-[var(--lp-border)] pb-3 text-sm text-[var(--lp-subtle)] sm:grid">
        <div>Instead of</div>
        <div>Qnetward</div>
      </div>
      <ul>
        {rows.map((row) => (
          <li
            key={row.before}
            className="grid gap-1 border-b border-[var(--lp-border)] py-5 text-base sm:grid-cols-2 sm:gap-8"
          >
            <div className="text-[var(--lp-subtle)]">{row.before}</div>
            <div className="text-[var(--lp-text)]">{row.after}</div>
          </li>
        ))}
      </ul>
    </div>
  );
}
