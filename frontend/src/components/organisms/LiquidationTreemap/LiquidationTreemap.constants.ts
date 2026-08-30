export const LIQ_TREEMAP_MAX_TILES = 80;
export const LIQ_TREEMAP_GUTTER = 2;

export const LIQ_TREEMAP_NEUTRAL = '#A1A7BB';
export const LIQ_TREEMAP_INK = '#FFFFFF';
export const LIQ_TREEMAP_INK_ON_NEUTRAL = '#0D1421';

/** Dominance (longShare) bands. High long share = red; high short share = green. */
export const LIQ_TREEMAP_BANDS = [
  { upTo: 0.15, fill: '#0B8F63' },
  { upTo: 0.3, fill: '#16C784' },
  { upTo: 0.42, fill: '#3FD39A' },
  { upTo: 0.48, fill: '#8EE8C4' },
  { upTo: 0.52, fill: LIQ_TREEMAP_NEUTRAL },
  { upTo: 0.58, fill: '#F5B4B8' },
  { upTo: 0.7, fill: '#F07B84' },
  { upTo: 0.85, fill: '#EA3943' },
  { upTo: Number.POSITIVE_INFINITY, fill: '#9B1B30' },
] as const;
