# 6. Marketing site and console in one Next.js app

## Status

Accepted

## Context

Qnetward needs a public landing page, legal pages, and auth pages that match
its brand, alongside the signed-in console. A separate marketing app would
mean a second deployment and duplicated design tokens.

## Decision

Keep everything in `apps/web`, split with route groups:

- `(marketing)`: `/`, `/terms`, `/privacy`, with their own layout and styles
  (`features/landing/landing.css`, scoped under `.lp-root`).
- `(auth)`: sign-in, sign-up, password reset, and verification, reusing the
  landing tokens plus `features/auth/auth.css` (scoped under `.lp-auth`).
- `(protected)`: the console. Its overview moved from `/` to `/overview` so
  the landing page can own `/`.

Brand styles are scoped by a root class rather than shared globals, so changes
to the marketing site can't restyle the console.

## Consequences

- Redirects after sign-in, sidebar links, and alert email links point to
  `/overview`.
- `globals.css` resets that would override utilities must sit in
  `@layer base`; unlayered rules beat Tailwind's layered utilities.
- If marketing ever needs its own release cycle, `(marketing)` can move to
  `apps/site` with little change.
