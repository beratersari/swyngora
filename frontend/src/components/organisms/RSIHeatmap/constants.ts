/** Neutral / missing RSI — desk ash. */
export const RSI_HEAT_NEUTRAL = '#A1A7BB';

/**
 * Same greens/reds as the price heatmap. Green = oversold, red = overbought.
 */
export const RSI_HEAT_BANDS = [
  { upTo: 20, fill: '#0EA872' },
  { upTo: 30, fill: '#16C784' },
  { upTo: 40, fill: '#3FD39A' },
  { upTo: 45, fill: '#8EE8C4' },
  { upTo: 55, fill: RSI_HEAT_NEUTRAL },
  { upTo: 60, fill: '#F5B4B8' },
  { upTo: 70, fill: '#F07B84' },
  { upTo: 80, fill: '#EA3943' },
  { upTo: Number.POSITIVE_INFINITY, fill: '#F6465D' },
] as const;

export const RSI_PLOT_PAD = { left: 52, right: 72, top: 20, bottom: 44 };

export const RSI_DOT_RADIUS = 6;

export const RSI_HEAT_INTERVALS = ['15m', '1h', '4h', '1d'] as const;

export const RSI_HEAT_TOPS = [50, 100] as const;
