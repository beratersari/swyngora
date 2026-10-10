export const GRID_METRICS = ['funding', 'oi', 'cvd', 'liquidations'] as const;
export type GridMetric = (typeof GRID_METRICS)[number];

export const GRID_RANGES = ['24h', '7d'] as const;
export type GridRange = (typeof GRID_RANGES)[number];

export const DEFAULT_GRID_METRIC: GridMetric = 'funding';
export const DEFAULT_GRID_RANGE: GridRange = '24h';

/** Funding of 0.05% per interval, a 5% open-interest move, or 4× a coin's own liquidation median is full color. */
export const GRID_FULL = {
  funding: 0.0005,
  oi: 5,
  liquidations: 3,
} as const;
