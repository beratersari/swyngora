import type { IndicatorsResponse } from '@/libs/api';

/** Lightweight Charts line point (UTCTimestamp seconds). */
export type ChartLinePoint = {
  time: number;
  value: number;
};

const DASH = '—';

/** Format an indicator number for display (RSI / EMA). */
export function formatIndicator(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return DASH;
  return value.toLocaleString(undefined, {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}

/**
 * RSI interpretation bands for UI coloring (not trading advice).
 * oversold &lt; 30 · neutral 30–70 · overbought &gt; 70
 */
export function rsiTone(
  value: number | null | undefined,
): 'success' | 'error' | 'warning' | 'secondary' {
  if (value === null || value === undefined || !Number.isFinite(value)) return 'secondary';
  if (value < 30) return 'success'; // oversold — often highlighted as opportunity (green)
  if (value > 70) return 'error'; // overbought
  if (value < 40 || value > 60) return 'warning';
  return 'secondary';
}

export function rsiBandLabel(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return 'n/a';
  if (value < 30) return 'oversold';
  if (value > 70) return 'overbought';
  return 'neutral';
}

/** Map indicator points → RSI line series for Lightweight Charts. */
export function indicatorPointsToRsiLine(
  points: IndicatorsResponse['points'] | undefined,
): ChartLinePoint[] {
  if (!points?.length) return [];
  const out: ChartLinePoint[] = [];
  for (const p of points) {
    if (!p.openTime || p.rsi === null || p.rsi === undefined || !Number.isFinite(p.rsi)) continue;
    const ms = Date.parse(p.openTime);
    if (!Number.isFinite(ms)) continue;
    out.push({ time: Math.floor(ms / 1000), value: p.rsi });
  }
  return out;
}

/** Map indicator points → EMA line series for a given period key (e.g. "12"). */
export function indicatorPointsToEmaLine(
  points: IndicatorsResponse['points'] | undefined,
  periodKey: string,
): ChartLinePoint[] {
  if (!points?.length) return [];
  const out: ChartLinePoint[] = [];
  for (const p of points) {
    const v = p.ema?.[periodKey];
    if (!p.openTime || v === undefined || !Number.isFinite(v)) continue;
    const ms = Date.parse(p.openTime);
    if (!Number.isFinite(ms)) continue;
    out.push({ time: Math.floor(ms / 1000), value: v });
  }
  return out;
}

/** Sorted EMA period keys present on latest snapshot. */
export function sortedEmaKeys(ema: Record<string, number> | undefined): string[] {
  if (!ema) return [];
  return Object.keys(ema).sort((a, b) => Number(a) - Number(b));
}
