/** Columns the heatmap can paint. Labels live in the derivatives locale. */
export const HEATMAP_METRICS = [
  'funding',
  'change1h',
  'change24h',
  'longPct',
  'takerBuyPct',
  'gapPct',
  'cvdDay',
  'cvdSplit',
  'printChange',
  'orderCount',
  'unusual',
  'liqHeat',
  'bidPct',
] as const;

export type HeatMetric = (typeof HEATMAP_METRICS)[number];

export const DEFAULT_HEATMAP_METRIC: HeatMetric = 'funding';

/**
 * Distance from neutral that paints a cell at full strength.
 * Day CVD is a notional, so its strength is this coin against the largest
 * day CVD on the screen. Price versus CVD uses 60 minutes as full strength,
 * so an in-line reading stays neutral. The other scales stay fixed so a quiet
 * hour does not paint every cell at full strength.
 */
export const HEATMAP_FULL: Record<HeatMetric, number | 'relative'> = {
  funding: 0.05,
  change1h: 5,
  change24h: 15,
  longPct: 20,
  takerBuyPct: 20,
  gapPct: 10,
  cvdDay: 'relative',
  cvdSplit: 3600,
  printChange: 80,
  orderCount: 0.5,
  unusual: 40,
  liqHeat: 3,
  bidPct: 20,
};
