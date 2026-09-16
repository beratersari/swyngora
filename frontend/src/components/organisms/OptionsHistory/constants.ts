export const OPTIONS_HISTORY_RANGES = ['12h', '48h', '7d'] as const;
export const OPTIONS_HISTORY_METRICS = ['oi', 'pcr', 'iv', 'gex', 'flip'] as const;

export const OPTIONS_HISTORY_RANGE_MS: Record<(typeof OPTIONS_HISTORY_RANGES)[number], number> = {
  '12h': 12 * 60 * 60 * 1000,
  '48h': 48 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
};
