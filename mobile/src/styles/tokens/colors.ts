/**
 * Swyngora brand color palette (source of truth).
 *
 * Primary / secondary / neutral from product design system.
 * Role aliases (`navy`, `indigo`, `steel`, `cream`) remain for existing
 * `theme.colors.*` call sites; prefer named palette keys in new code.
 */

/** Primary colors */
export const primaryColors = {
  /** Rich Black — deepest canvas / app chrome */
  richBlack: '#000F0F',
  /** Dark Green — primary surface / page background */
  darkGreen: '#032221',
  /** Bangladesh Green — primary actions, strong borders */
  bangladeshGreen: '#03624C',
  /** Mountain Meadow — accents, chart up, positive */
  mountainMeadow: '#4FD4A5',
  /** Caribbean Green — high-emphasis accent / focus */
  caribbeanGreen: '#00FF81',
  /** Anti-Flash White — primary text on dark */
  antiFlashWhite: '#F1F7F6',
} as const;

/** Secondary greens (elevated surfaces, depth) */
export const secondaryColors = {
  pine: '#063028',
  basil: '#0B453A',
  forest: '#095544',
  frog: '#17876D',
  mint: '#74F9BC',
} as const;

/** Neutrals */
export const neutralColors = {
  stone: '#707D7D',
  pistachio: '#AACBC4',
} as const;

/** Full named palette */
export const palette = {
  ...primaryColors,
  ...secondaryColors,
  ...neutralColors,
} as const;

export type PaletteColorName = keyof typeof palette;

/**
 * Role aliases used by theme / legacy components.
 * Prefer `palette.*` or `semanticColors` in new code.
 */
export const colors = {
  ...palette,

  /** @deprecated Prefer `richBlack` / `darkGreen` — canvas chrome */
  navy: primaryColors.richBlack,
  /** @deprecated Prefer `bangladeshGreen` — primary action / elevated */
  indigo: primaryColors.bangladeshGreen,
  /** @deprecated Prefer `stone` — secondary text */
  steel: neutralColors.stone,
  /** @deprecated Prefer `antiFlashWhite` — primary text */
  cream: primaryColors.antiFlashWhite,
} as const;

export type BrandColorName = keyof typeof colors;

/** Semantic aliases mapped from the brand palette */
export const semanticColors = {
  bg: {
    canvas: palette.richBlack,
    elevated: palette.basil,
    muted: palette.darkGreen,
    inverse: palette.antiFlashWhite,
  },
  text: {
    primary: palette.antiFlashWhite,
    secondary: neutralColors.stone,
    inverse: palette.richBlack,
    link: palette.mountainMeadow,
    disabled: 'rgba(112, 125, 125, 0.55)', // stone @ 55%
  },
  border: {
    default: 'rgba(170, 203, 196, 0.28)', // pistachio soft
    strong: palette.bangladeshGreen,
    focus: palette.caribbeanGreen,
  },
  action: {
    primary: palette.bangladeshGreen,
    primaryHover: palette.frog,
    primaryText: palette.antiFlashWhite,
  },
  status: {
    success: palette.mountainMeadow,
    warning: '#E0B86A', // keep readable warning (no brand yellow in set)
    error: '#E07A7A', // keep readable error (no brand red in set)
    info: neutralColors.pistachio,
  },
  chart: {
    up: palette.caribbeanGreen,
    down: '#E07A7A',
    grid: 'rgba(170, 203, 196, 0.18)',
    background: palette.richBlack,
    text: neutralColors.stone,
    emaFast: palette.mountainMeadow,
    emaSlow: palette.frog,
    rsi: palette.antiFlashWhite,
  },
  skeleton: {
    base: 'rgba(11, 69, 58, 0.55)', // basil
    highlight: 'rgba(241, 247, 246, 0.1)', // anti-flash soft
  },
} as const;

export type SemanticColors = typeof semanticColors;
