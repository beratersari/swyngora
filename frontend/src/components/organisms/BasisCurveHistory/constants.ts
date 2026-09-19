export const BASIS_CURVE_RANGES = ['24h', '7d'] as const;
export const BASIS_CURVE_WINDOWS = ['1h', '4h', '24h', '7d'] as const;

export const BASIS_CURVE_RANGE_MS: Record<(typeof BASIS_CURVE_RANGES)[number], number> = {
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
};
