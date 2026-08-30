export const LIQ_BAR_PAD = { left: 72, right: 20, top: 16, bottom: 36 } as const;
export const LIQ_BAR_LONG = '#EA3943';
export const LIQ_BAR_SHORT = '#16C784';
export const LIQ_BAR_SPINE = '#C5CDD8';
export const LIQ_BAR_LAST = '#3861FB';
export const LIQ_BAR_INK = '#616E85';
export const LIQ_BAR_PLOT = '#FFFFFF';
export const LIQ_LEVELS_POLL_MS = 60_000;
export const LIQ_CHART_RANGES = ['12h', '24h'] as const;

/** CoinGlass-style leverage colors: 100x orange, 50x yellow, 25x blue, 10x light blue. */
export const LIQ_LEVERAGE_ORDER = [100, 50, 25, 10] as const;
export const LIQ_LEVERAGE_COLOR: Record<(typeof LIQ_LEVERAGE_ORDER)[number], string> = {
  100: '#F7931A',
  50: '#F5C042',
  25: '#3B82F6',
  10: '#93C5FD',
};
