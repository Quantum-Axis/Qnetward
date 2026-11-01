/**
 * The artwork beside the auth form: broad, softly lit ribbons in the brand
 * colours, like signal traces swept across the panel. Pure SVG, no image file.
 */
export function AuthArt() {
  return (
    <div className="auth-art relative h-full w-full overflow-hidden bg-[#121215]" aria-hidden="true">
      <svg viewBox="0 0 600 760" preserveAspectRatio="xMidYMid slice" className="absolute inset-0 h-full w-full">
        <defs>
          <linearGradient id="auth-ribbon-a" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor="#ff2e74" />
            <stop offset="55%" stopColor="#ff4f6d" />
            <stop offset="100%" stopColor="#ff8a6b" />
          </linearGradient>
          <linearGradient id="auth-ribbon-b" x1="0" y1="1" x2="1" y2="0">
            <stop offset="0%" stopColor="#c0265e" />
            <stop offset="60%" stopColor="#ff3d7f" />
            <stop offset="100%" stopColor="#ff9b8a" />
          </linearGradient>
          <linearGradient id="auth-ribbon-c" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.9" />
            <stop offset="100%" stopColor="#ff3d7f" stopOpacity="0.9" />
          </linearGradient>
          <linearGradient id="auth-sheen" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#fff" stopOpacity="0.55" />
            <stop offset="100%" stopColor="#fff" stopOpacity="0" />
          </linearGradient>
          <radialGradient id="auth-vignette" cx="70%" cy="35%" r="80%">
            <stop offset="45%" stopColor="#121215" stopOpacity="0" />
            <stop offset="100%" stopColor="#121215" stopOpacity="0.92" />
          </radialGradient>
          <filter id="auth-soft" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="18" />
          </filter>
          <filter id="auth-edge" x="-10%" y="-10%" width="120%" height="120%">
            <feGaussianBlur stdDeviation="1.2" />
          </filter>
        </defs>

        {/* glow beneath the ribbons */}
        <g filter="url(#auth-soft)" opacity="0.55">
          <path d="M-80 250 C 160 120, 360 220, 680 60" stroke="#ff3d7f" strokeWidth="160" fill="none" />
          <path d="M-60 560 C 200 420, 420 520, 700 380" stroke="#ff6b6b" strokeWidth="140" fill="none" />
        </g>

        <g className="auth-art-drift">
          {/* back ribbon */}
          <path d="M-120 520 C 120 380, 300 470, 720 300" stroke="url(#auth-ribbon-c)" strokeWidth="70" strokeLinecap="round" fill="none" opacity="0.55" />
          {/* main sweep */}
          <path d="M-100 180 C 140 40, 380 160, 720 -20" stroke="url(#auth-ribbon-a)" strokeWidth="150" strokeLinecap="round" fill="none" />
          <path d="M-100 180 C 140 40, 380 160, 720 -20" stroke="url(#auth-sheen)" strokeWidth="150" strokeLinecap="round" fill="none" opacity="0.18" />
          {/* lower sweep */}
          <path d="M-140 470 C 120 300, 360 420, 720 250" stroke="url(#auth-ribbon-b)" strokeWidth="130" strokeLinecap="round" fill="none" />
          <path d="M-140 470 C 120 300, 360 420, 720 250" stroke="url(#auth-sheen)" strokeWidth="130" strokeLinecap="round" fill="none" opacity="0.14" />
          {/* thin highlights along the ribbon edges */}
          <path d="M-100 112 C 140 -28, 380 92, 720 -88" stroke="#ffc2d4" strokeWidth="2" fill="none" opacity="0.55" filter="url(#auth-edge)" />
          <path d="M-140 408 C 120 238, 360 358, 720 188" stroke="#ffd0c4" strokeWidth="2" fill="none" opacity="0.45" filter="url(#auth-edge)" />
          <path d="M120 470 C 260 420, 420 470, 720 420" stroke="#ffffff" strokeWidth="1.5" fill="none" opacity="0.18" />
        </g>

        <rect width="600" height="760" fill="url(#auth-vignette)" />
      </svg>
    </div>
  );
}
