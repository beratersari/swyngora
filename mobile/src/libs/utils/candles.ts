/** Candle shape from Swyngora API (OHLCV fields are strings). */
export type ApiCandle = {
  openTime: string;
  open: string;
  high: string;
  low: string;
  close: string;
  volume?: string;
  closeTime?: string;
};

/** Lightweight Charts candlestick bar (time as UTCTimestamp seconds). */
export type ChartCandle = {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
};

export type ChartLinePoint = {
  time: number;
  value: number;
};

/** Map API candles to lightweight-charts candlestick data. */
export function apiCandlesToChart(candles: ApiCandle[]): ChartCandle[] {
  return candles
    .map((c) => {
      const openTimeMs = Date.parse(c.openTime);
      const open = Number(c.open);
      const high = Number(c.high);
      const low = Number(c.low);
      const close = Number(c.close);
      if (
        !Number.isFinite(openTimeMs) ||
        !Number.isFinite(open) ||
        !Number.isFinite(high) ||
        !Number.isFinite(low) ||
        !Number.isFinite(close)
      ) {
        return null;
      }
      return {
        time: Math.floor(openTimeMs / 1000),
        open,
        high,
        low,
        close,
      };
    })
    .filter((x): x is ChartCandle => x !== null);
}

/**
 * Merge candle series by open time (seconds). Later arguments win on conflict.
 * Result is sorted ascending (oldest → newest).
 */
export function mergeChartCandles(...series: ChartCandle[][]): ChartCandle[] {
  const map = new Map<number, ChartCandle>();
  for (const list of series) {
    for (const c of list) {
      map.set(c.time, c);
    }
  }
  return [...map.values()].sort((a, b) => a.time - b.time);
}

/**
 * endTime for fetching bars strictly before the oldest chart candle.
 * Returns RFC3339 (preferred by exchanges) with 1ms subtracted from open.
 */
export function endTimeBeforeOldestCandle(
  oldest: ChartCandle | undefined,
): string | undefined {
  if (!oldest || !Number.isFinite(oldest.time)) return undefined;
  const ms = oldest.time * 1000 - 1;
  if (!Number.isFinite(ms) || ms <= 0) return undefined;
  return new Date(ms).toISOString();
}

export function sortedEmaKeys(ema: Record<string, number> | undefined): string[] {
  if (!ema) return [];
  return Object.keys(ema).sort((a, b) => Number(a) - Number(b));
}

const MIN_EMA_PERIOD = 2;

/** Parse `"12,26"` into unique ascending periods. */
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

/** EMA over closes. Seed is SMA of the first `period` closes. */
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

/** EMA line on the same time axis as loaded chart candles. */
export function emaLineFromCloses(
  bars: ReadonlyArray<{ time: number; close: number }>,
  period: number,
): ChartLinePoint[] {
  if (!bars.length) return [];
  const ema = emaFromCloses(
    bars.map((b) => b.close),
    period,
  );
  const out: ChartLinePoint[] = [];
  for (let i = 0; i < ema.length; i += 1) {
    const v = ema[i];
    if (v == null || !Number.isFinite(v) || !Number.isFinite(bars[i]!.time)) continue;
    out.push({ time: bars[i]!.time, value: v });
  }
  return out;
}

export function indicatorPointsToEmaLine(
  points:
    | {
        openTime?: string;
        ema?: Record<string, number>;
      }[]
    | undefined,
  periodKey: string,
): ChartLinePoint[] {
  if (!points?.length) return [];
  const out: ChartLinePoint[] = [];
  for (const p of points) {
    if (!p.openTime || !p.ema || p.ema[periodKey] === undefined) continue;
    const t = Date.parse(p.openTime);
    const v = Number(p.ema[periodKey]);
    if (!Number.isFinite(t) || !Number.isFinite(v)) continue;
    out.push({ time: Math.floor(t / 1000), value: v });
  }
  return out;
}

export function indicatorPointsToRsi(
  points:
    | {
        openTime?: string;
        rsi?: number | null;
      }[]
    | undefined,
): ChartLinePoint[] {
  if (!points?.length) return [];
  const out: ChartLinePoint[] = [];
  for (const p of points) {
    if (!p.openTime || p.rsi === null || p.rsi === undefined) continue;
    const t = Date.parse(p.openTime);
    const v = Number(p.rsi);
    if (!Number.isFinite(t) || !Number.isFinite(v)) continue;
    out.push({ time: Math.floor(t / 1000), value: v });
  }
  return out;
}

export function resolveInterval(
  preferred: string | undefined,
  supported: string[] | undefined,
  fallback = '1h',
): string {
  if (supported?.length) {
    if (preferred && supported.includes(preferred)) return preferred;
    if (supported.includes(fallback)) return fallback;
    return supported[0];
  }
  return preferred || fallback;
}

export const EMA_LINE_COLORS = ['#4FD4A5', '#74F9BC', '#00FF81', '#17876D'] as const;

export function emaColor(_periodKey: string, index: number): string {
  return EMA_LINE_COLORS[index % EMA_LINE_COLORS.length] ?? '#4FD4A5';
}
