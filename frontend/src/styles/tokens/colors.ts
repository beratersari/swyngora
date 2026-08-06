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

/** Hex → rgba helper for token-only alpha variants (no ad-hoc hex in components). */
export function withAlpha(hex: string, alpha: number): string {
  const h = hex.replace('#', '');
  const full =
    h.length === 3
      ? h
          .split('')
          .map((c) => c + c)
          .join('')
      : h;
  const n = Number.parseInt(full, 16);
  const r = (n >> 16) & 255;
  const g = (n >> 8) & 255;
  const b = n & 255;
  const a = Math.min(1, Math.max(0, alpha));
  return `rgba(${r}, ${g}, ${b}, ${a})`;
}

/**
 * Semantic aliases mapped from the brand palette.
 *
 * Role rules (professional dark UI):
 * - Surfaces: canvas (richest black) → muted cards (darkGreen) → elevated (basil) → chrome (pine)
 * - UI accents / links / active: mountainMeadow (not neon caribbean)
 * - caribbeanGreen: chart “up” + focus ring only
 * - Secondary text: pistachio (readable); stone = tertiary / muted chrome
 */
export const semanticColors = {
  bg: {
    /** App shell / deepest canvas */
    canvas: palette.richBlack,
    /** Page background under content (slightly lifted from canvas) */
    page: palette.darkGreen,
    /** Cards, tables, panels */
    muted: palette.darkGreen,
    /** Header / chrome bars */
    chrome: palette.pine,
    /** Popovers, dropdowns, elevated panels */
    elevated: palette.basil,
    /** Inverse (light) surfaces */
    inverse: palette.antiFlashWhite,
    /** Row / control hover */
    hover: withAlpha(palette.frog, 0.22),
    /** Table header band */
    tableHeader: withAlpha(palette.bangladeshGreen, 0.42),
    /** Soft accent wash (badges, selected chips) */
    accentSoft: withAlpha(palette.mountainMeadow, 0.12),
    /** User chat bubble / selected surface */
    accentMuted: palette.forest,
    /** Error surface */
    dangerSoft: withAlpha('#E07A7A', 0.16),
  },
  text: {
    primary: palette.antiFlashWhite,
    /** Labels, captions — pistachio for contrast on dark greens */
    secondary: neutralColors.pistachio,
    /** Meta / disabled-looking chrome */
    tertiary: neutralColors.stone,
    inverse: palette.richBlack,
    link: palette.mountainMeadow,
    linkHover: palette.mint,
    disabled: withAlpha(neutralColors.stone, 0.55),
    accent: palette.mountainMeadow,
  },
  border: {
    default: withAlpha(neutralColors.pistachio, 0.28),
    subtle: withAlpha(neutralColors.pistachio, 0.16),
    strong: palette.bangladeshGreen,
    /** High-emphasis focus only */
    focus: palette.caribbeanGreen,
    /** Soft accent outline (badges, selected) */
    accent: withAlpha(palette.mountainMeadow, 0.55),
    danger: withAlpha('#E07A7A', 0.65),
  },
  action: {
    primary: palette.bangladeshGreen,
    primaryHover: palette.frog,
    primaryActive: palette.forest,
    primaryText: palette.antiFlashWhite,
    /** Secondary / ghost button border */
    secondaryBorder: withAlpha(neutralColors.pistachio, 0.4),
  },
  /** UI accent (tabs, badges, active nav) — not neon chart green */
  accent: {
    default: palette.mountainMeadow,
    soft: withAlpha(palette.mountainMeadow, 0.14),
    strong: palette.frog,
  },
  status: {
    success: palette.mountainMeadow,
    warning: '#E0B86A', // readable warning (no brand yellow in set)
    error: '#E07A7A', // readable error (no brand red in set)
    info: neutralColors.pistachio,
  },
  chart: {
    /** Neon reserved for price direction / focus, not chrome */
    up: palette.caribbeanGreen,
    down: '#E07A7A',
    grid: withAlpha(neutralColors.pistachio, 0.16),
    background: palette.richBlack,
    text: neutralColors.pistachio,
    emaFast: palette.mountainMeadow,
    emaSlow: palette.frog,
    rsi: palette.antiFlashWhite,
  },
  skeleton: {
    base: withAlpha(palette.basil, 0.85),
    mid: withAlpha(palette.forest, 0.9),
    highlight: withAlpha(palette.mint, 0.18),
  },
} as const;

export type SemanticColors = typeof semanticColors;
