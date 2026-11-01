import Link from "next/link";
import { ArrowRight, Check } from "lucide-react";
import { horizonMeta, itemsByHorizon, type Horizon } from "./roadmap-data";

const dot: Record<Horizon, string> = {
  now: "var(--lp-green)",
  next: "var(--lp-accent)",
  later: "var(--lp-subtle)",
};

/** Now / Next / Later at a glance on the landing page; the details live on /roadmap. */
export function RoadmapPreview() {
  const columns: Horizon[] = ["now", "next", "later"];
  return (
    <div className="lp-reveal">
      <div className="flex items-baseline justify-between gap-6 border-b border-[var(--lp-border)] pb-4">
        <h3 className="text-[15px] font-medium">Roadmap</h3>
        <Link href="/roadmap" className="inline-flex items-center gap-1.5 text-[13px] text-[var(--lp-muted)] hover:text-[var(--lp-text)]">
          Full roadmap <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
        </Link>
      </div>
      <div className="grid md:grid-cols-3">
        {columns.map((horizon, i) => {
          const items = itemsByHorizon(horizon);
          return (
            <section key={horizon} className={`border-b border-[var(--lp-border)] py-6 md:border-b-0 ${i > 0 ? "md:border-l md:pl-8" : ""} md:pr-8`}>
              <div className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full" style={{ background: dot[horizon] }} aria-hidden="true" />
                <h4 className="text-[14px] font-medium">{horizonMeta[horizon].label}</h4>
                <span className="text-[13px] text-[var(--lp-subtle)]">· {horizonMeta[horizon].caption}</span>
                <span className="lp-mono ml-auto rounded-md border border-[var(--lp-border)] px-1.5 text-[11.5px] text-[var(--lp-subtle)]">{items.length}</span>
              </div>
              <ul className="mt-4 space-y-2.5">
                {items.map((item) => (
                  <li key={item.name} className={`flex items-center gap-2 text-[14px] leading-snug ${horizon === "now" ? "text-[var(--lp-text)]" : "text-[var(--lp-muted)]"}`}>
                    {horizon === "now" ? (
                      <Check className="h-3.5 w-3.5 shrink-0 text-[var(--lp-green)]" aria-hidden="true" />
                    ) : (
                      <span className="h-3.5 w-3.5 shrink-0 text-center leading-[14px] text-[var(--lp-subtle)]" aria-hidden="true">
                        ·
                      </span>
                    )}
                    {item.name}
                  </li>
                ))}
              </ul>
            </section>
          );
        })}
      </div>
    </div>
  );
}
