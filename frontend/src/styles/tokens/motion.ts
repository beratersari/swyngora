/**
 * Motion tokens — desk micro-interactions.
 * Keep durations short; trading UIs should feel snappy, not cinematic.
 * Always honor `prefers-reduced-motion` at the call site / GlobalStyle.
 */
export const motionDuration = {
  /** Hover color / border */
  instant: '80ms',
  /** Buttons, chips, nav pills */
  fast: '140ms',
  /** Page enter, star pop, card lift */
  base: '220ms',
  /** Price flash, live pulse fade */
  slow: '640ms',
} as const;

export const motionEase = {
  /** Default UI (Material-ish emphasized decelerate) */
  standard: 'cubic-bezier(0.2, 0, 0, 1)',
  enter: 'cubic-bezier(0.16, 1, 0.3, 1)',
  exit: 'cubic-bezier(0.4, 0, 1, 1)',
} as const;

export const motion = {
  duration: motionDuration,
  ease: motionEase,
} as const;

export type MotionDurationName = keyof typeof motionDuration;
export type MotionEaseName = keyof typeof motionEase;
