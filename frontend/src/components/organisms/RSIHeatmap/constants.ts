/** Midpoint of the scale (RSI 50) and missing readings. */
export const RSI_HEAT_NEUTRAL = '#A1A7BB';

/**
 * Interpolation stops for a continuous green → ash → red RSI scale.
 * Green = oversold, red = overbought (same family as the price heatmap).
 */
export const RSI_HEAT_STOPS = [
  { at: 0, fill: '#0EA872' },
  { at: 20, fill: '#16C784' },
  { at: 30, fill: '#3FD39A' },
  { at: 40, fill: '#8EE8C4' },
  { at: 50, fill: RSI_HEAT_NEUTRAL },
  { at: 60, fill: '#F5B4B8' },
  { at: 70, fill: '#F07B84' },
  { at: 80, fill: '#EA3943' },
  { at: 100, fill: '#F6465D' },
] as const;

export const RSI_PLOT_PAD = { left: 52, right: 72, top: 20, bottom: 44 };

export const RSI_DOT_RADIUS = 6;

/** Max distance from a dot center (plot units ≈ CSS px) to show its card. */
export const RSI_HOVER_REACH = 18;

export const RSI_TIP = { w: 200, h: 78 } as const;

export const RSI_HEAT_INTERVALS = ['15m', '1h', '4h', '1d'] as const;

export const RSI_HEAT_TOPS = [50, 100] as const;
