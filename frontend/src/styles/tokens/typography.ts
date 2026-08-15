/**
 * Swyngora type scale (source of truth).
 * Font stacks are system-first for performance; brand feel comes from scale + color.
 */
export const fontFamilies = {
  sans: 'Inter, "Segoe UI", system-ui, -apple-system, sans-serif',
  mono: 'Inter, "Segoe UI", system-ui, -apple-system, sans-serif',
} as const;

export const fontWeights = {
  regular: 400,
  medium: 500,
  semibold: 600,
  bold: 700,
} as const;

/**
 * Named text roles used by the Text atom and layout.
 * size = font-size px, lineHeight = unitless or px string for CSS.
 */
export const typeScale = {
  display: {
    fontSize: 32,
    lineHeight: 1.15,
    fontWeight: fontWeights.semibold,
    letterSpacing: '-0.025em',
  },
  h1: {
    fontSize: 26,
    lineHeight: 1.2,
    fontWeight: fontWeights.semibold,
    letterSpacing: '-0.02em',
  },
  h2: {
    fontSize: 20,
    lineHeight: 1.25,
    fontWeight: fontWeights.semibold,
    letterSpacing: '-0.015em',
  },
  h3: {
    fontSize: 18,
    lineHeight: 1.3,
    fontWeight: fontWeights.semibold,
    letterSpacing: '-0.01em',
  },
  h4: {
    fontSize: 16,
    lineHeight: 1.4,
    fontWeight: fontWeights.semibold,
    letterSpacing: '0',
  },
  bodyLg: {
    fontSize: 16,
    lineHeight: 1.55,
    fontWeight: fontWeights.regular,
    letterSpacing: '0',
  },
  body: {
    fontSize: 14,
    lineHeight: 1.55,
    fontWeight: fontWeights.regular,
    letterSpacing: '0',
  },
  bodySm: {
    fontSize: 13,
    lineHeight: 1.5,
    fontWeight: fontWeights.regular,
    letterSpacing: '0',
  },
  caption: {
    fontSize: 12,
    lineHeight: 1.4,
    fontWeight: fontWeights.regular,
    letterSpacing: '0.01em',
  },
  overline: {
    fontSize: 11,
    lineHeight: 1.35,
    fontWeight: fontWeights.semibold,
    letterSpacing: '0.08em',
    textTransform: 'uppercase' as const,
  },
  label: {
    fontSize: 13,
    lineHeight: 1.35,
    fontWeight: fontWeights.medium,
    letterSpacing: '0.01em',
  },
  code: {
    fontSize: 13,
    lineHeight: 1.45,
    fontWeight: fontWeights.regular,
    letterSpacing: '0',
    fontFamily: fontFamilies.mono,
  },
  /** Tabular figures for prices / volumes */
  numeric: {
    fontSize: 14,
    lineHeight: 1.4,
    fontWeight: fontWeights.medium,
    letterSpacing: '0',
    fontFamily: fontFamilies.mono,
    fontVariantNumeric: 'tabular-nums' as const,
  },
} as const;

export type TypeVariant = keyof typeof typeScale;

export const textColors = {
  primary: 'primary',
  secondary: 'secondary',
  inverse: 'inverse',
  cream: 'cream',
  steel: 'steel',
  success: 'success',
  warning: 'warning',
  error: 'error',
} as const;

export type TextColor = keyof typeof textColors;
