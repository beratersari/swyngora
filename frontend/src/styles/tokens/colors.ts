/**
 * Swyngora market-desk palette.
 *
 * Institutional charcoal + amber (Bloomberg / terminal), not neon green.
 * Legacy key names (`bangladeshGreen`, `mountainMeadow`, …) stay as aliases
 * so existing call sites pick up the new hex values.
 */

export const financeColors = {
  ink: '#07090D',
  graphite: '#0C1016',
  slate: '#141922',
  steel: '#1B2130',
  pewter: '#252C3A',
  ivory: '#F3F5F8',
  silver: '#A3ABB8',
  mist: '#7A8393',
  ash: '#5A6270',
  amber: '#F0A202',
  amberHover: '#FFC14A',
  amberDeep: '#C4840C',
  up: '#2BB673',
  down: '#E5534B',
  warning: '#E8B339',
  cobalt: '#5B8DEF',
} as const;

/** Primary colors (legacy names → finance hex). */
export const primaryColors = {
  richBlack: financeColors.ink,
  darkGreen: financeColors.graphite,
  bangladeshGreen: financeColors.amber,
  mountainMeadow: financeColors.amber,
  caribbeanGreen: financeColors.up,
  antiFlashWhite: financeColors.ivory,
} as const;

/** Secondary surfaces / accent ramps (legacy names). */
export const secondaryColors = {
  pine: financeColors.slate,
  basil: financeColors.steel,
  forest: financeColors.pewter,
  frog: financeColors.amberDeep,
  mint: financeColors.amberHover,
} as const;

/** Neutrals */
export const neutralColors = {
  stone: financeColors.ash,
  sage: financeColors.mist,
  pistachio: financeColors.silver,
} as const;

export const palette = {
  ...financeColors,
  ...primaryColors,
  ...secondaryColors,
  ...neutralColors,
} as const;

export type PaletteColorName = keyof typeof palette;

export const colors = {
  ...palette,
  /** @deprecated Prefer `ink` / `richBlack` */
  navy: financeColors.ink,
  /** @deprecated Prefer `amber` */
  indigo: financeColors.amber,
  /** @deprecated Prefer `ash` / `stone` */
  steel: financeColors.ash,
  /** @deprecated Prefer `ivory` */
  cream: financeColors.ivory,
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
 * Semantic roles. Amber is brand / chrome. Green/red are market direction only.
 */
export const semanticColors = {
  bg: {
    canvas: financeColors.ink,
    page: financeColors.graphite,
    muted: financeColors.slate,
    chrome: financeColors.steel,
    elevated: financeColors.pewter,
    inverse: financeColors.ivory,
    hover: withAlpha(financeColors.amber, 0.08),
    tableHeader: financeColors.steel,
    accentSoft: withAlpha(financeColors.amber, 0.12),
    accentMuted: financeColors.pewter,
    dangerSoft: withAlpha(financeColors.down, 0.14),
    successSoft: withAlpha(financeColors.up, 0.14),
  },
  text: {
    primary: financeColors.ivory,
    secondary: financeColors.silver,
    tertiary: financeColors.mist,
    inverse: financeColors.ink,
    link: financeColors.amber,
    linkHover: financeColors.amberHover,
    disabled: withAlpha(financeColors.mist, 0.55),
    accent: financeColors.amber,
  },
  border: {
    default: withAlpha(financeColors.ivory, 0.1),
    subtle: withAlpha(financeColors.ivory, 0.06),
    strong: withAlpha(financeColors.ivory, 0.18),
    focus: financeColors.amber,
    accent: withAlpha(financeColors.amber, 0.7),
    danger: withAlpha(financeColors.down, 0.65),
  },
  action: {
    primary: financeColors.amber,
    primaryHover: financeColors.amberHover,
    primaryActive: financeColors.amberDeep,
    primaryText: financeColors.ink,
    secondaryBorder: withAlpha(financeColors.ivory, 0.16),
  },
  accent: {
    default: financeColors.amber,
    soft: withAlpha(financeColors.amber, 0.14),
    strong: financeColors.amberDeep,
  },
  status: {
    success: financeColors.up,
    warning: financeColors.warning,
    error: financeColors.down,
    info: financeColors.cobalt,
  },
  chart: {
    up: financeColors.up,
    down: financeColors.down,
    neutral: '#3A4250',
    mapBed: financeColors.ink,
    grid: withAlpha(financeColors.ivory, 0.08),
    background: financeColors.ink,
    text: financeColors.silver,
    emaFast: financeColors.amber,
    emaSlow: financeColors.cobalt,
    rsi: financeColors.ivory,
  },
  skeleton: {
    base: withAlpha(financeColors.steel, 0.9),
    mid: withAlpha(financeColors.pewter, 0.9),
    highlight: withAlpha(financeColors.amber, 0.14),
  },
} as const;

export type SemanticColors = typeof semanticColors;
