/** Cap tiles to the realtime WS max (docs/features/realtime.md: 100 per connection). */
export const HEATMAP_MAX_TILES = 100;

/** Visible white street between tiles. 2px matches CoinMarketCap (1px inset each side). */
export const HEATMAP_GUTTER = 2;

/** |change %| treated as unchanged (true dead band). Stables stay gray. */
export const HEATMAP_DEAD_ZONE_PCT = 0.05;

/** |change %| at which the scale saturates (legend labels). */
export const HEATMAP_COLOR_CAP_PCT = 10;

/** Neutral / flat tile — CMC slate gray. */
export const HEATMAP_NEUTRAL = '#A1A7BB';

/** Tile label ink. CMC uses white on every colored cell. */
export const HEATMAP_TILE_INK = '#FFFFFF';

export const HEATMAP_TILE_INK_ON_NEUTRAL = '#0D1421';

/**
 * Discrete CoinMarketCap-style bands. First `upTo` that the change is ≤ wins.
 * Saturated greens/reds so white labels stay readable.
 */
export const HEATMAP_BANDS = [
  { upTo: -10, fill: '#9B1B30' },
  { upTo: -5, fill: '#EA3943' },
  { upTo: -2, fill: '#F6465D' },
  { upTo: -0.5, fill: '#F07B84' },
  { upTo: -HEATMAP_DEAD_ZONE_PCT, fill: '#F5B4B8' },
  { upTo: HEATMAP_DEAD_ZONE_PCT, fill: HEATMAP_NEUTRAL },
  { upTo: 0.5, fill: '#8EE8C4' },
  { upTo: 2, fill: '#3FD39A' },
  { upTo: 5, fill: '#16C784' },
  { upTo: 10, fill: '#0EA872' },
  { upTo: Number.POSITIVE_INFINITY, fill: '#0B8F63' },
] as const;

export const HEATMAP_LEGEND_GRADIENT = HEATMAP_BANDS.map((band, i, arr) => {
  const start = i === 0 ? -HEATMAP_COLOR_CAP_PCT : arr[i - 1]!.upTo;
  const end = Number.isFinite(band.upTo) ? band.upTo : HEATMAP_COLOR_CAP_PCT;
  const pct = ((start + HEATMAP_COLOR_CAP_PCT) / (HEATMAP_COLOR_CAP_PCT * 2)) * 100;
  const pctEnd = ((end + HEATMAP_COLOR_CAP_PCT) / (HEATMAP_COLOR_CAP_PCT * 2)) * 100;
  return `${band.fill} ${Math.max(0, pct).toFixed(1)}% ${Math.min(100, pctEnd).toFixed(1)}%`;
}).join(', ');
