/**
 * Shared layout breakpoints (desktop-first desk, mobile wrap).
 * Use `media.*` in styled-components; use the same px in `useMediaQuery`.
 */
export const breakpoints = {
  /** Narrow phones */
  xs: 480,
  /** Phones / small fold */
  phone: 720,
  /** Tablets / small laptops */
  tablet: 960,
} as const;

export type BreakpointName = keyof typeof breakpoints;

export const media = {
  xs: `@media (max-width: ${breakpoints.xs - 0.02}px)`,
  phone: `@media (max-width: ${breakpoints.phone - 0.02}px)`,
  tablet: `@media (max-width: ${breakpoints.tablet - 0.02}px)`,
} as const;

export const mediaQueries = {
  xs: `(max-width: ${breakpoints.xs - 0.02}px)`,
  phone: `(max-width: ${breakpoints.phone - 0.02}px)`,
  tablet: `(max-width: ${breakpoints.tablet - 0.02}px)`,
} as const;
