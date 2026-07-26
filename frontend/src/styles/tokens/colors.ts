/**
 * Swyngora brand color palette (source of truth).
 * Also mirrored as CSS variables in tokens.css.
 */
export const colors = {
  /** Deep navy — app background, header, primary surface */
  navy: '#111844',
  /** Indigo — elevated surfaces, borders, secondary actions */
  indigo: '#4B5694',
  /** Steel — muted text, icons, secondary labels */
  steel: '#7288AE',
  /** Cream — primary text on dark, accents, highlights */
  cream: '#EAE0CF',
} as const;

export type BrandColorName = keyof typeof colors;

/** Semantic aliases mapped from the brand palette */
export const semanticColors = {
  bg: {
    canvas: colors.navy,
    elevated: colors.indigo,
    muted: '#1a2250', // navy lifted slightly for cards (derived)
    inverse: colors.cream,
  },
  text: {
    primary: colors.cream,
    secondary: colors.steel,
    inverse: colors.navy,
    link: colors.cream,
    disabled: 'rgba(114, 136, 174, 0.55)',
  },
  border: {
    default: 'rgba(114, 136, 174, 0.35)',
    strong: colors.indigo,
    focus: colors.cream,
  },
  action: {
    primary: colors.indigo,
    primaryHover: '#5c68a8',
    primaryText: colors.cream,
  },
  status: {
    // Keep brand-adjacent status hues for trading UI
    success: '#6fbf8a',
    warning: '#e0b86a',
    error: '#e07a7a',
    info: colors.steel,
  },
  chart: {
    up: '#6fbf8a',
    down: '#e07a7a',
    grid: 'rgba(114, 136, 174, 0.2)',
    background: colors.navy,
    text: colors.steel,
  },
  skeleton: {
    base: 'rgba(75, 86, 148, 0.45)',
    highlight: 'rgba(234, 224, 207, 0.12)',
  },
} as const;

export type SemanticColors = typeof semanticColors;
