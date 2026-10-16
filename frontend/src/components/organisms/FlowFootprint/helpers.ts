import type { FlowAtPricesLevel, FlowAtPricesReport, FlowAtPricesVenue } from '@/libs/api';
import { IMBALANCE_RATIO, ONE_SIDED_SHARE } from './FlowFootprint.constants';

export function flowNum(value?: string | number): number {
  const n = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function footprintVenue(report?: FlowAtPricesReport | null): FlowAtPricesVenue | undefined {
  if (!report) return undefined;
  if ((report.combined?.levels?.length ?? 0) > 0) return report.combined;
  // A late venue keeps the combined ladder off. One venue's rows are not the whole window.
  if (report.exchange === 'all' && report.definite === false) return undefined;
  return report.venues?.find((v) => (v.levels?.length ?? 0) > 0 && !v.error) ?? report.venues?.[0];
}

export function imbalanceRatio(row: FlowAtPricesLevel): number {
  const buy = flowNum(row.buyVolume);
  const sell = flowNum(row.sellVolume);
  const hi = Math.max(buy, sell);
  const lo = Math.min(buy, sell);
  if (hi <= 0) return 1;
  if (lo <= 0) return Number.POSITIVE_INFINITY;
  return hi / lo;
}

/** A row is a strong imbalance when one side is at least 3× the other. */
export function isStrongImbalance(row: FlowAtPricesLevel): boolean {
  const buy = flowNum(row.buyVolume);
  const sell = flowNum(row.sellVolume);
  if (buy + sell <= 0) return false;
  const ratio = imbalanceRatio(row);
  if (!Number.isFinite(ratio)) return flowNum(row.sharePct) >= ONE_SIDED_SHARE;
  return ratio >= IMBALANCE_RATIO;
}

export function imbalanceSide(row: FlowAtPricesLevel): 'buy' | 'sell' | null {
  if (!isStrongImbalance(row)) return null;
  return flowNum(row.buyVolume) >= flowNum(row.sellVolume) ? 'buy' : 'sell';
}

/** Highest price first, matching the API ladder. */
export function footprintRows(levels?: FlowAtPricesLevel[]): FlowAtPricesLevel[] {
  return (levels ?? [])
    .filter((row) => flowNum(row.volume) > 0 || flowNum(row.buyVolume) + flowNum(row.sellVolume) > 0)
    .slice()
    .sort((a, b) => flowNum(b.price) - flowNum(a.price));
}

export function sidePeak(rows: FlowAtPricesLevel[]): number {
  let peak = 0;
  for (const row of rows) {
    peak = Math.max(peak, flowNum(row.buyVolume), flowNum(row.sellVolume));
  }
  return peak;
}

export function barPct(value: string | undefined, peak: number): number {
  if (peak <= 0) return 0;
  return Math.max(0, Math.min(100, (flowNum(value) / peak) * 100));
}
