"use client";

import { useEffect } from "react";

/**
 * Page-wide progressive enhancements, attached once:
 * - `.lp-reveal` elements fade up when scrolled into view (hidden only after JS runs,
 *   so content stays visible without JavaScript).
 * - `.lp-spot` cards get a pointer-following highlight via --mx / --my.
 */
export function LandingEffects() {
  useEffect(() => {
    const root = document.querySelector(".lp-root");
    if (!root) return;

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let io: IntersectionObserver | undefined;
    if (!reduced) {
      root.classList.add("lp-reveal-ready");
      io = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (entry.isIntersecting) {
              entry.target.classList.add("lp-in");
              io?.unobserve(entry.target);
            }
          }
        },
        { rootMargin: "0px 0px -10% 0px", threshold: 0.08 },
      );
      root.querySelectorAll(".lp-reveal").forEach((el) => io?.observe(el));
    }

    const onMove = (e: PointerEvent) => {
      const card = (e.target as HTMLElement | null)?.closest<HTMLElement>(".lp-spot");
      if (!card) return;
      const rect = card.getBoundingClientRect();
      card.style.setProperty("--mx", `${e.clientX - rect.left}px`);
      card.style.setProperty("--my", `${e.clientY - rect.top}px`);
    };
    root.addEventListener("pointermove", onMove as EventListener, { passive: true });

    return () => {
      io?.disconnect();
      root.removeEventListener("pointermove", onMove as EventListener);
    };
  }, []);

  return null;
}
