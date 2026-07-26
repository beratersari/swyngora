/** Candle shape from Swyngora API (OHLCV fields are strings). */
export type ApiCandle = {
  openTime: string;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: string;
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
