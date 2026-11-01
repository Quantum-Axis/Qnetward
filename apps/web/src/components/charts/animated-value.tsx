"use client";

import { useEffect, useRef, useState } from "react";

function parse(value: string) {
  const match = value.match(/^(-?\d+(?:\.\d+)?)(.*)$/);
  if (!match) return null;
  const decimals = match[1].includes(".") ? match[1].split(".")[1].length : 0;
  return { num: Number.parseFloat(match[1]), decimals, suffix: match[2] };
}

/**
 * Renders a value like "58.8ms" or "12" and tweens from the previous number to
 * the new one. A brief green or red tint shows whether the change was good.
 * Non-numeric values (e.g. "5 / 5", "—") render as-is.
 */
export function AnimatedValue({ value, better }: { value: string; better?: "lower" | "higher" }) {
  // `tween` holds the in-between text while animating; otherwise the real value renders.
  const [tween, setTween] = useState<string | null>(null);
  const [flash, setFlash] = useState<{ key: string; tone: "good" | "bad" } | null>(null);
  const prevRef = useRef<number | null>(null);

  useEffect(() => {
    const target = parse(value);
    const from = prevRef.current;
    prevRef.current = target ? target.num : null;
    if (!target || from === null || from === target.num || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const started = performance.now();
    let raf = 0;
    let first = true;
    const step = (now: number) => {
      if (first && better) setFlash({ key: value, tone: (target.num < from) === (better === "lower") ? "good" : "bad" });
      first = false;
      const k = Math.min(1, (now - started) / 450);
      const eased = 1 - Math.pow(1 - k, 3);
      setTween(k < 1 ? `${(from + (target.num - from) * eased).toFixed(target.decimals)}${target.suffix}` : null);
      if (k < 1) raf = requestAnimationFrame(step);
    };
    raf = requestAnimationFrame(step);
    return () => cancelAnimationFrame(raf);
  }, [value, better]);

  const tone = flash?.key === value ? flash.tone : null;
  return (
    <span key={tone ? `${value}-${tone}` : value} className={tone === "good" ? "lp-value-good" : tone === "bad" ? "lp-value-bad" : undefined}>
      {tween ?? value}
    </span>
  );
}
