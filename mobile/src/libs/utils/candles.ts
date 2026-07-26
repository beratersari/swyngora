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

export function sortedEmaKeys(ema: Record<string, number> | undefined): string[] {
  if (!ema) return [];
  return Object.keys(ema).sort((a, b) => Number(a) - Number(b));
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
