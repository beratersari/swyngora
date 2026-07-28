import type { ChartCandle } from '@/libs/utils';
import type { CandleChartMarker } from './CandleChartHost.types';

export type SnapMarkersOptions = {
  /**
   * If set, drop markers whose nearest candle is farther than this many seconds.
   * Prevents events outside the loaded window from all stacking on the left edge.
   */
  maxDistanceSec?: number;
};

/**
 * Lightweight Charts only draws markers whose `time` exists in series data.
 * Pump API openTimes can differ slightly from candle openTimes (ms/nano, boundary).
 * Snap each marker to the nearest bar time so zoom/pan never drops markers.
 */
export function snapMarkersToCandleTimes(
  markers: CandleChartMarker[],
  candles: ChartCandle[],
  options?: SnapMarkersOptions,
): CandleChartMarker[] {
  if (!markers.length || !candles.length) return [];

  const times = candles.map((c) => c.time).filter((t) => Number.isFinite(t));
  if (!times.length) return [];

  const maxDist = options?.maxDistanceSec;
  // times are ascending from API; keep a Set for O(1) exact hits
  const timeSet = new Set(times);
  const out: CandleChartMarker[] = [];
  const used = new Set<number>();

  for (const m of markers) {
    if (!Number.isFinite(m.time)) continue;
    let t = m.time;
    if (!timeSet.has(t)) {
      t = nearestTime(times, m.time);
    }
    if (
      maxDist != null &&
      Number.isFinite(maxDist) &&
      maxDist >= 0 &&
      Math.abs(t - m.time) > maxDist
    ) {
      continue;
    }
    // One marker per bar (keep first / stronger already ordered by caller)
    if (used.has(t)) continue;
    used.add(t);
    out.push({ ...m, time: t });
  }
  return out;
}

/** Binary search nearest bar time in a sorted ascending array. */
export function nearestTime(sortedTimes: number[], target: number): number {
  if (!sortedTimes.length) return target;
  let lo = 0;
  let hi = sortedTimes.length - 1;
  if (target <= sortedTimes[0]!) return sortedTimes[0]!;
  if (target >= sortedTimes[hi]!) return sortedTimes[hi]!;

  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    const v = sortedTimes[mid]!;
    if (v === target) return v;
    if (v < target) lo = mid + 1;
    else hi = mid - 1;
  }
  // lo is first > target, hi is last < target
  const a = sortedTimes[hi]!;
  const b = sortedTimes[lo]!;
  return Math.abs(a - target) <= Math.abs(b - target) ? a : b;
}
