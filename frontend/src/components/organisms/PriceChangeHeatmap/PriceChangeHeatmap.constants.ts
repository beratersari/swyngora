import { semanticColors } from '@/styles/tokens';

/** Cap tiles so the map stays readable (API spot max is 500). */
export const HEATMAP_MAX_TILES = 80;

/** Hairline seam; must match the map bed so it recedes. */
export const HEATMAP_GUTTER = 1;

/** |change %| at which the scale saturates. */
export const HEATMAP_COLOR_CAP_PCT = 8;

/** |change %| treated as unchanged (true dead band). */
export const HEATMAP_DEAD_ZONE_PCT = 0.4;

export const HEATMAP_BED = semanticColors.chart.mapBed;
export const HEATMAP_NEUTRAL = semanticColors.chart.neutral;

/**
 * Opaque diverging scale. 0% is cool slate (not green).
 * Near-zero stops stay dark so the mosaic sits on the charcoal well.
 */
export const HEATMAP_SCALE = [
  { at: -8, fill: '#EA3943' },
  { at: -4, fill: '#F37A81' },
  { at: -1.5, fill: '#F8B4B8' },
  { at: 0, fill: HEATMAP_NEUTRAL },
  { at: 1.5, fill: '#9BE8C8' },
  { at: 4, fill: '#3ED39A' },
  { at: 8, fill: '#16C784' },
] as const;

export const HEATMAP_LEGEND_GRADIENT = HEATMAP_SCALE.map((s, i, arr) => {
  const pct = ((s.at + HEATMAP_COLOR_CAP_PCT) / (HEATMAP_COLOR_CAP_PCT * 2)) * 100;
  return `${s.fill} ${pct.toFixed(1)}%${i === arr.length - 1 ? '' : ''}`;
}).join(', ');
