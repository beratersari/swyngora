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

/** Stable band key for i18n (`detail:indicators.band.*`). */
export type RsiBandKey = 'na' | 'oversold' | 'overbought' | 'neutral';

/** Map the API `zone` field (Go `RSIZoneFor`) onto UI color. */
export function rsiTone(
  zone: string | null | undefined,
): 'success' | 'error' | 'warning' | 'secondary' {
  switch (zone) {
    case 'oversold':
      return 'success';
    case 'overbought':
      return 'error';
    default:
      return 'secondary';
  }
}

/** Map the API `zone` field onto an i18n band key. */
export function rsiBandKey(zone: string | null | undefined): RsiBandKey {
  if (zone === 'oversold' || zone === 'overbought' || zone === 'neutral') return zone;
  return 'na';
}

/** English band label (tests / non-UI). Prefer rsiBandKey + t() in components. */
export function rsiBandLabel(zone: string | null | undefined): string {
  const key = rsiBandKey(zone);
  if (key === 'na') return 'n/a';
  return key;
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

export type IndicatorPoint = NonNullable<IndicatorsResponse['points']>[number];

/** Merge older + newer indicator pages by openTime (newer wins). */
export function mergeIndicatorPoints(
  older: readonly IndicatorPoint[] | undefined,
  newer: readonly IndicatorPoint[] | undefined,
): IndicatorPoint[] {
  const byTime = new Map<string, IndicatorPoint>();
  for (const p of older ?? []) {
    const key = p.openTime?.trim();
    if (key) byTime.set(key, p);
  }
  for (const p of newer ?? []) {
    const key = p.openTime?.trim();
    if (key) byTime.set(key, p);
  }
  return [...byTime.values()].sort((a, b) => {
    const am = Date.parse(a.openTime ?? '');
    const bm = Date.parse(b.openTime ?? '');
    return am - bm;
  });
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
