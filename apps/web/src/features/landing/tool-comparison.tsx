import { Clock, Globe, Laptop, SquareTerminal, type AppIcon } from "@/components/icons";
import { QnetwardMark } from "./qnetward-mark";

// The four tabs Qnetward replaces, placed round it. x/y are the circle's
// centre as a percentage of the diagram.
const tools: { icon: AppIcon; name: string; x: number; y: number }[] = [
  { icon: SquareTerminal, name: "ping and mtr", x: 16, y: 20 },
  { icon: Laptop, name: "nmap on a laptop", x: 16, y: 80 },
  { icon: Globe, name: "dig and a DNS site", x: 84, y: 20 },
  { icon: Clock, name: "An uptime checker", x: 84, y: 80 },
];

/** Five tabs into one: the old tools round the edge, Qnetward in the middle. */
export function ToolComparison() {
  return (
    <div className="lp-reveal text-center">
      <h2 className="mx-auto max-w-3xl text-3xl font-semibold leading-[1.12] tracking-[-0.03em] sm:text-[44px]">
        <span className="text-[var(--lp-text)]">Five tabs during an outage</span>
        <br />
        <span className="text-[var(--lp-muted)]">is four too many</span>
      </h2>

      <div className="relative mx-auto mt-12 aspect-[5/3] w-full max-w-[720px] sm:mt-16 sm:aspect-[2/1]">
        {/* A soft glow behind the middle. */}
        <div
          className="pointer-events-none absolute left-1/2 top-1/2 h-[90%] w-[60%] -translate-x-1/2 -translate-y-1/2 rounded-full"
          style={{ background: "radial-gradient(closest-side, rgba(255,61,127,0.22), transparent)" }}
          aria-hidden="true"
        />

        {/* Dashed lines from each tool into the middle. */}
        <svg className="absolute inset-0 h-full w-full" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
          {tools.map((t) => (
            <line
              key={t.name}
              x1={t.x}
              y1={t.y}
              x2={50}
              y2={50}
              stroke="rgba(255,255,255,0.12)"
              strokeWidth={1}
              strokeDasharray="3 4"
              vectorEffect="non-scaling-stroke"
            />
          ))}
        </svg>

        <ul>
          {tools.map((t) => (
            <li
              key={t.name}
              className="absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center"
              style={{ left: `${t.x}%`, top: `${t.y}%` }}
            >
              <span className="grid h-12 w-12 place-items-center rounded-full border border-[rgba(255,61,127,0.22)] bg-[var(--lp-bg-raised)] sm:h-[72px] sm:w-[72px]">
                <t.icon className="h-5 w-5 text-[var(--lp-muted)] sm:h-6 sm:w-6" aria-hidden="true" />
              </span>
              <span className="absolute top-full mt-2 whitespace-nowrap text-xs text-[var(--lp-subtle)] sm:text-sm">{t.name}</span>
            </li>
          ))}
        </ul>

        <div className="absolute left-1/2 top-1/2 grid h-20 w-20 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full border-[1.5px] border-[var(--lp-accent)] bg-[var(--lp-bg-raised)] shadow-[0_0_60px_rgba(255,61,127,0.35)] sm:h-32 sm:w-32">
          <span className="sm:hidden">
            <QnetwardMark size={32} />
          </span>
          <span className="hidden sm:block">
            <QnetwardMark size={48} />
          </span>
          <span className="sr-only">Qnetward</span>
        </div>
      </div>

      <p className="mx-auto mt-14 max-w-xl text-md leading-relaxed text-[var(--lp-muted)] sm:mt-16">
        Most teams stitch diagnostics together from terminal commands, a website or two, and a separate uptime service.
        Qnetward puts them behind one login, with shared history.
      </p>
      <a
        href="#products"
        className="lp-btn mt-7 h-11 bg-[var(--lp-text)] px-5 text-md font-medium text-[#0b0b0d] hover:bg-white"
      >
        See every tool <span aria-hidden="true">→</span>
      </a>
    </div>
  );
}
