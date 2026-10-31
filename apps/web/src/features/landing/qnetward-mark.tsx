/** Qnetward logo mark: a "Q" ring with a probe signal crossing its tail. */
export function QnetwardMark({ size = 24 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <rect width="32" height="32" rx="8" fill="#ff3d7f" />
      <circle cx="15" cy="15" r="7.5" stroke="#fff" strokeWidth="2.6" />
      <path d="M19.5 19.5 25 25" stroke="#fff" strokeWidth="2.6" strokeLinecap="round" />
      <circle cx="15" cy="15" r="2.2" fill="#fff" />
    </svg>
  );
}
