/**
 * Consumer market-site palette (CoinMarketCap-like).
 * Light canvas, blue brand, green/red only for price direction.
 * Legacy key names stay as aliases so older call sites keep compiling.
 */

export const financeColors = {
  ink: '#0D1421',
  paper: '#FFFFFF',
  mistBg: '#F8FAFD',
  chip: '#EFF2F5',
  line: '#CFD6E4',
  ivory: '#0D1421',
  silver: '#616E85',
  mist: '#8B95A7',
  ash: '#A1A7BB',
  amber: '#3861FB',
  amberHover: '#6188FF',
  amberDeep: '#1E53E5',
  up: '#16C784',
  down: '#EA3943',
  warning: '#F5B544',
  cobalt: '#3861FB',
  graphite: '#F8FAFD',
  slate: '#FFFFFF',
  steel: '#FFFFFF',
  pewter: '#FFFFFF',
} as const;

export const primaryColors = {
  richBlack: financeColors.ink,
  darkGreen: financeColors.mistBg,
  bangladeshGreen: financeColors.amber,
  mountainMeadow: financeColors.amber,
  caribbeanGreen: financeColors.up,
  antiFlashWhite: financeColors.ink,
} as const;

export const secondaryColors = {
  pine: financeColors.chip,
  basil: financeColors.paper,
  forest: financeColors.chip,
  frog: financeColors.amberDeep,
  mint: financeColors.amberHover,
} as const;

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
  navy: financeColors.ink,
  indigo: financeColors.amber,
  steel: financeColors.silver,
  cream: financeColors.ink,
} as const;

export type BrandColorName = keyof typeof colors;

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

export const semanticColors = {
  bg: {
    canvas: financeColors.paper,
    page: financeColors.mistBg,
    muted: financeColors.paper,
    chrome: financeColors.paper,
    elevated: financeColors.paper,
    inverse: financeColors.ink,
    hover: withAlpha(financeColors.amber, 0.06),
    tableHeader: financeColors.mistBg,
    accentSoft: withAlpha(financeColors.amber, 0.1),
    accentMuted: financeColors.chip,
    dangerSoft: withAlpha(financeColors.down, 0.1),
    successSoft: withAlpha(financeColors.up, 0.1),
  },
  text: {
    primary: financeColors.ink,
    secondary: financeColors.silver,
    tertiary: financeColors.mist,
    inverse: financeColors.paper,
    link: financeColors.amber,
    linkHover: financeColors.amberDeep,
    disabled: withAlpha(financeColors.mist, 0.55),
    accent: financeColors.amber,
  },
  border: {
    default: financeColors.chip,
    subtle: financeColors.chip,
    strong: financeColors.line,
    focus: financeColors.amber,
    accent: withAlpha(financeColors.amber, 0.45),
    danger: withAlpha(financeColors.down, 0.45),
  },
  action: {
    primary: financeColors.amber,
    primaryHover: financeColors.amberHover,
    primaryActive: financeColors.amberDeep,
    primaryText: financeColors.paper,
    secondaryBorder: financeColors.line,
  },
  accent: {
    default: financeColors.amber,
    soft: withAlpha(financeColors.amber, 0.1),
    strong: financeColors.amberDeep,
  },
  status: {
    success: financeColors.up,
    warning: financeColors.warning,
    error: financeColors.down,
    info: financeColors.amber,
  },
  chart: {
    up: financeColors.up,
    down: financeColors.down,
    neutral: '#C7CDD8',
    mapBed: '#E9EEF5',
    grid: withAlpha(financeColors.ink, 0.08),
    background: financeColors.paper,
    text: financeColors.silver,
    emaFast: financeColors.amber,
    emaSlow: '#16C784',
    rsi: financeColors.ink,
  },
  skeleton: {
    base: financeColors.chip,
    mid: financeColors.line,
    highlight: withAlpha(financeColors.amber, 0.16),
  },
} as const;

export type SemanticColors = typeof semanticColors;
