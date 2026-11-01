import { ReactNode } from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { AuthArt } from "@/components/auth/auth-art";
import { QnetwardMark } from "@/features/landing/qnetward-mark";

type HeaderAction = { prompt: string; label: string; href: string };

/**
 * Split auth card: the form on the left, artwork on the right (hidden on small
 * screens). The logo heads the form, the other path (sign in / sign up) sits
 * under it, and the terms line sits under the card.
 */
export function AuthLayoutCard({
  title,
  subtitle,
  headerAction,
  wide = false,
  contentClassName,
  below,
  children,
}: {
  title: string;
  subtitle: ReactNode;
  /** Kept for compatibility; the split card has no icon badge. */
  icon?: ReactNode;
  /** Kept for compatibility; no longer rendered. */
  eyebrow?: string;
  headerAction?: HeaderAction;
  /** Wider form column, used by the multi-step sign-up. */
  wide?: boolean;
  contentClassName?: string;
  /** Shown under the card, above the terms line (e.g. the demo entry). */
  below?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-[#18181b] px-4 py-8 sm:py-12">
      <div
        className={cn(
          "grid w-full overflow-hidden rounded-2xl border border-[var(--lp-border)] bg-[#1d1d21] shadow-[0_24px_80px_rgba(0,0,0,0.45)] lg:grid-cols-2",
          wide ? "max-w-[960px]" : "max-w-[900px]",
        )}
      >
        <div className="flex flex-col px-6 py-9 sm:px-10 sm:py-11">
          <Link href="/" className="mb-8 flex w-fit items-center gap-2" aria-label="Qnetward home">
            <QnetwardMark size={22} />
            <span className="text-[15px] font-semibold tracking-tight">Qnetward</span>
          </Link>
          <h1 className="text-[24px] font-semibold tracking-[-0.02em]">{title}</h1>
          <div className="mt-1.5 text-[14px] leading-relaxed text-[var(--lp-muted)]">{subtitle}</div>
          <div className={cn("mt-7 flex-1", contentClassName)}>{children}</div>
          {headerAction ? (
            <p className="mt-7 text-center text-[14px] text-[var(--lp-muted)]">
              {headerAction.prompt}{" "}
              <Link href={headerAction.href} className="auth-footer-link">
                {headerAction.label}
              </Link>
            </p>
          ) : null}
        </div>
        <div className="relative hidden min-h-[560px] lg:block">
          <AuthArt />
        </div>
      </div>

      {below ? <div className="mt-6">{below}</div> : null}
      <p className="mt-6 text-center text-[12px] text-[var(--lp-subtle)]">
        By continuing, you agree to our{" "}
        <Link href="/terms" className="auth-footer-link text-[var(--lp-muted)]">
          Terms of Service
        </Link>{" "}
        and{" "}
        <Link href="/privacy" className="auth-footer-link text-[var(--lp-muted)]">
          Privacy Policy
        </Link>
        .
      </p>
    </div>
  );
}
