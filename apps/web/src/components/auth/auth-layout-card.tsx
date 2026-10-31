import { ReactNode } from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { NetworkCanvas } from "@/features/landing/network-canvas";
import { QnetwardMark } from "@/features/landing/qnetward-mark";

type HeaderAction = { prompt: string; label: string; href: string };

/** Concentric "radar" rings behind the card. */
function RadarRings() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 1200 1200"
      className="lp-radar pointer-events-none absolute left-1/2 top-1/2 h-[1400px] w-[1400px] -translate-x-1/2 -translate-y-1/2"
    >
      <defs>
        <radialGradient id="lp-ring-fade" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#fff" stopOpacity="0.16" />
          <stop offset="70%" stopColor="#fff" stopOpacity="0.05" />
          <stop offset="100%" stopColor="#fff" stopOpacity="0" />
        </radialGradient>
      </defs>
      {[180, 280, 390, 510].map((r) => (
        <circle key={r} cx="600" cy="600" r={r} fill="none" stroke="url(#lp-ring-fade)" strokeWidth="1" />
      ))}
      <circle cx="600" cy="600" r="180" fill="none" stroke="#ff3d7f" strokeOpacity="0.5" strokeWidth="1" className="lp-radar-pulse" />
    </svg>
  );
}

/**
 * Centered auth card inside a framed stage. The secondary path (sign in / sign up)
 * sits in the frame's top-right corner so the card has one primary action.
 */
export function AuthLayoutCard({
  title,
  subtitle,
  icon,
  headerAction,
  wide = false,
  contentClassName,
  children,
}: {
  title: string;
  subtitle: ReactNode;
  icon?: ReactNode;
  /** Kept for compatibility; no longer rendered. */
  eyebrow?: string;
  headerAction?: HeaderAction;
  /** Wider card, used by the multi-step sign-up. */
  wide?: boolean;
  contentClassName?: string;
  children: ReactNode;
}) {
  return (
    <div className="min-h-screen bg-[#0d0d10] p-3 sm:p-5">
      <div className="relative flex min-h-[calc(100vh-24px)] flex-col overflow-hidden rounded-[22px] border border-[var(--lp-border)] bg-[#141417] sm:min-h-[calc(100vh-40px)] sm:rounded-[28px]">
        {/* stage background */}
        <div className="pointer-events-none absolute inset-0">
          <div
            className="absolute inset-0"
            style={{
              background:
                "radial-gradient(ellipse 60% 45% at 50% 0%, rgba(255,61,127,0.16), transparent 70%), radial-gradient(ellipse 50% 40% at 50% 100%, rgba(124,58,237,0.12), transparent 70%)",
            }}
          />
          <NetworkCanvas className="absolute inset-0 h-full w-full opacity-30 [mask-image:radial-gradient(ellipse_at_center,transparent_18%,black_45%,transparent_85%)]" />
          <RadarRings />
        </div>

        <header className="relative z-10 flex items-center justify-between gap-4 px-6 py-5 sm:px-10 sm:py-7">
          <Link href="/" className="flex items-center gap-2.5" aria-label="Qnetward home">
            <QnetwardMark size={26} />
            <span className="text-[16px] font-semibold tracking-tight">Qnetward</span>
          </Link>
          {headerAction ? (
            <p className="text-[14px] text-[var(--lp-muted)]">
              <span className="hidden sm:inline">{headerAction.prompt} </span>
              <Link href={headerAction.href} className="auth-footer-link">
                {headerAction.label}
              </Link>
            </p>
          ) : null}
        </header>

        <main className="relative z-10 flex flex-1 items-center justify-center px-4 py-8">
          <div className={cn("auth-card w-full", wide ? "max-w-[460px]" : "max-w-[400px]")}>
            {icon ? (
              <div className="auth-card-icon mx-auto" aria-hidden="true">
                {icon}
              </div>
            ) : null}
            <h1 className="mt-5 text-center text-[22px] font-medium tracking-[-0.015em]">{title}</h1>
            <div className="mx-auto mt-2 max-w-[320px] text-balance text-center text-[14px] leading-relaxed text-[var(--lp-muted)]">{subtitle}</div>
            <div className={cn("mt-7", contentClassName)}>{children}</div>
          </div>
        </main>

        <footer className="relative z-10 flex flex-wrap justify-center gap-x-5 gap-y-2 px-6 pb-6 text-[12px] text-[var(--lp-subtle)]">
          <Link href="/" className="hover:text-[var(--lp-text)]">
            Home
          </Link>
          <Link href="/terms" className="hover:text-[var(--lp-text)]">
            Terms
          </Link>
          <Link href="/privacy" className="hover:text-[var(--lp-text)]">
            Privacy
          </Link>
          <span>Free during beta</span>
        </footer>
      </div>
    </div>
  );
}
