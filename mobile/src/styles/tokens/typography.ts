/**
 * Swyngora type scale — sizes/weights aligned with frontend tokens.
 * Font stacks are system-first for React Native Web + native later.
 */
export const fontFamilies = {
  sans: 'system-ui, -apple-system, "Segoe UI", Roboto, sans-serif',
  mono: 'ui-monospace, "SF Mono", Menlo, Consolas, monospace',
} as const;

export const fontWeights = {
  regular: '400' as const,
  medium: '500' as const,
  semibold: '600' as const,
  bold: '700' as const,
};

export const typeScale = {
  display: { fontSize: 36, lineHeight: 42, fontWeight: fontWeights.bold },
  h1: { fontSize: 30, lineHeight: 36, fontWeight: fontWeights.bold },
  h2: { fontSize: 24, lineHeight: 30, fontWeight: fontWeights.semibold },
  h3: { fontSize: 20, lineHeight: 26, fontWeight: fontWeights.semibold },
  h4: { fontSize: 16, lineHeight: 22, fontWeight: fontWeights.semibold },
  bodyLg: { fontSize: 16, lineHeight: 24, fontWeight: fontWeights.regular },
  body: { fontSize: 14, lineHeight: 22, fontWeight: fontWeights.regular },
  bodySm: { fontSize: 13, lineHeight: 20, fontWeight: fontWeights.regular },
  caption: { fontSize: 12, lineHeight: 16, fontWeight: fontWeights.regular },
  overline: { fontSize: 11, lineHeight: 14, fontWeight: fontWeights.semibold },
  label: { fontSize: 13, lineHeight: 18, fontWeight: fontWeights.medium },
  code: { fontSize: 13, lineHeight: 18, fontWeight: fontWeights.regular },
  numeric: { fontSize: 14, lineHeight: 20, fontWeight: fontWeights.medium },
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
