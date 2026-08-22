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
 * Aligned with {@link rsiBandKey} so color and band label never disagree.
 */
export function rsiTone(
  value: number | null | undefined,
): 'success' | 'error' | 'warning' | 'secondary' {
  if (value === null || value === undefined || !Number.isFinite(value)) return 'secondary';
  if (value < 30) return 'success'; // oversold — often highlighted as opportunity (green)
  if (value > 70) return 'error'; // overbought
  return 'secondary'; // neutral 30–70 (includes near-band values)
}

/** Stable band key for i18n (`detail:indicators.band.*`). */
export type RsiBandKey = 'na' | 'oversold' | 'overbought' | 'neutral';

export function rsiBandKey(value: number | null | undefined): RsiBandKey {
  if (value === null || value === undefined || !Number.isFinite(value)) return 'na';
  if (value < 30) return 'oversold';
  if (value > 70) return 'overbought';
  return 'neutral';
}

/** English band label (tests / non-UI). Prefer rsiBandKey + t() in components. */
export function rsiBandLabel(value: number | null | undefined): string {
  const key = rsiBandKey(value);
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

const MIN_EMA_PERIOD = 2;

/** Parse `"12,26"` into unique ascending periods (same rules as the backend). */
export function parseEmaPeriods(csv: string | undefined): number[] {
  if (!csv?.trim()) return [];
  const seen = new Set<number>();
  const out: number[] = [];
  for (const part of csv.split(',')) {
    const n = Number(part.trim());
    if (!Number.isInteger(n) || n < MIN_EMA_PERIOD || seen.has(n)) continue;
    seen.add(n);
    out.push(n);
  }
  return out.sort((a, b) => a - b);
}

/**
 * EMA over closes. Seed is SMA of the first `period` closes (matches Go `domain.EMA`).
 * Warm-up bars are `null`.
 */
export function emaFromCloses(closes: readonly number[], period: number): Array<number | null> {
  const n = closes.length;
  const out: Array<number | null> = Array.from({ length: n }, () => null);
  if (!Number.isInteger(period) || period < MIN_EMA_PERIOD || n < period) return out;
  let sum = 0;
  for (let i = 0; i < period; i += 1) sum += closes[i]!;
  let ema = sum / period;
  out[period - 1] = ema;
  const k = 2 / (period + 1);
  for (let i = period; i < n; i += 1) {
    ema = closes[i]! * k + ema * (1 - k);
    out[i] = ema;
  }
  return out;
}

/** EMA line on the same time axis as loaded chart candles (follows pan-left history). */
export function emaLineFromCloses(
  bars: ReadonlyArray<{ time: number; close: number }>,
  period: number,
): ChartLinePoint[] {
  if (!bars.length) return [];
  const closes = bars.map((b) => b.close);
  const ema = emaFromCloses(closes, period);
  const out: ChartLinePoint[] = [];
  for (let i = 0; i < ema.length; i += 1) {
    const v = ema[i];
    if (v == null || !Number.isFinite(v) || !Number.isFinite(bars[i]!.time)) continue;
    out.push({ time: bars[i]!.time, value: v });
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
