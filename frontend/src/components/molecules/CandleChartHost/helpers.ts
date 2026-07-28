import type { CandlestickData, LineData, Time } from 'lightweight-charts';
import type { ChartCandle, ChartLinePoint } from '@/libs/utils';
import type { CandleChartOverlay } from './CandleChartHost.types';

/** Lightweight Charts `priceFormat` for the price scale / crosshair labels. */
export type ChartPriceFormat = {
  type: 'price';
  precision: number;
  minMove: number;
};

const DEFAULT_CHART_PRICE_FORMAT: ChartPriceFormat = {
  type: 'price',
  precision: 2,
  minMove: 0.01,
};

/** Cap precision so labels stay readable; LWC handles small minMove via base. */
export const MAX_CHART_PRICE_PRECISION = 12;
export const MIN_CHART_PRICE_PRECISION = 2;

/**
 * How many fractional digits are needed to express a positive magnitude.
 * e.g. 123.4 → 2, 0.0000123 → 8 (rough order-of-magnitude).
 */
export function decimalsForMagnitude(value: number): number {
  const abs = Math.abs(value);
  if (!Number.isFinite(abs) || abs === 0) return MIN_CHART_PRICE_PRECISION;
  if (abs >= 1000) return 2;
  if (abs >= 1) return 4;
  if (abs >= 0.1) return 5;
  if (abs >= 0.01) return 6;
  if (abs >= 0.0001) return 8;
  if (abs >= 1e-6) return 10;
  if (abs >= 1e-8) return 12;
  return MAX_CHART_PRICE_PRECISION;
}

/**
 * Derive chart price precision from OHLC so micro-priced coins
 * (e.g. 0.00000xxxx) show movement on the axis instead of rounding to 0.00.
 */
export function chartPriceFormatFromCandles(data: ChartCandle[]): ChartPriceFormat {
  if (!data.length) return DEFAULT_CHART_PRICE_FORMAT;

  let minAbs = Number.POSITIVE_INFINITY;
  let maxAbs = 0;
  let minLow = Number.POSITIVE_INFINITY;
  let maxHigh = Number.NEGATIVE_INFINITY;

  for (const bar of data) {
    for (const p of [bar.open, bar.high, bar.low, bar.close]) {
      if (!Number.isFinite(p)) continue;
      const abs = Math.abs(p);
      if (abs > 0 && abs < minAbs) minAbs = abs;
      if (abs > maxAbs) maxAbs = abs;
    }
    if (Number.isFinite(bar.low) && bar.low < minLow) minLow = bar.low;
    if (Number.isFinite(bar.high) && bar.high > maxHigh) maxHigh = bar.high;
  }

  if (!Number.isFinite(minAbs) || minAbs === Number.POSITIVE_INFINITY) {
    return DEFAULT_CHART_PRICE_FORMAT;
  }

  let precision = decimalsForMagnitude(minAbs);

  // Ensure the visible high–low range is not flattened to a single tick.
  const range = maxHigh - minLow;
  if (Number.isFinite(range) && range > 0) {
    const step = range / 200; // ~200 distinct levels across the window
    const rangeDigits = Math.ceil(-Math.log10(step));
    if (Number.isFinite(rangeDigits)) {
      precision = Math.max(precision, rangeDigits);
    }
  }

  precision = Math.min(
    MAX_CHART_PRICE_PRECISION,
    Math.max(MIN_CHART_PRICE_PRECISION, Math.floor(precision)),
  );
  const minMove = 10 ** -precision;

  return { type: 'price', precision, minMove };
}

/**
 * Map UI candles → Lightweight Charts candlestick series points.
 * Sorts by time and drops duplicate timestamps (library requires strictly ascending times).
 */
export function toCandlestickData(data: ChartCandle[]): CandlestickData<Time>[] {
  if (!data.length) return [];
  const sorted = [...data]
    .filter(
      (d) =>
        Number.isFinite(d.time) &&
        Number.isFinite(d.open) &&
        Number.isFinite(d.high) &&
        Number.isFinite(d.low) &&
        Number.isFinite(d.close),
    )
    .sort((a, b) => a.time - b.time);
  const out: CandlestickData<Time>[] = [];
  let lastTime: number | null = null;
  for (const d of sorted) {
    const point: CandlestickData<Time> = {
      time: d.time as Time,
      open: d.open,
      high: d.high,
      low: d.low,
      close: d.close,
    };
    if (lastTime !== null && d.time === lastTime) {
      // Keep the later bar for the same timestamp (matches toLineData).
      out[out.length - 1] = point;
      continue;
    }
    out.push(point);
    lastTime = d.time;
  }
  return out;
}

/**
 * Map overlay line points → Lightweight Charts line data.
 * Sorts by time and drops duplicates (library requires strictly ascending times).
 */
export function toLineData(points: ChartLinePoint[]): LineData<Time>[] {
  if (!points.length) return [];
  const sorted = [...points]
    .filter((p) => Number.isFinite(p.time) && Number.isFinite(p.value))
    .sort((a, b) => a.time - b.time);
  const out: LineData<Time>[] = [];
  let lastTime: number | null = null;
  for (const p of sorted) {
    if (lastTime !== null && p.time === lastTime) {
      // Keep the later value for the same timestamp.
      out[out.length - 1] = { time: p.time as Time, value: p.value };
      continue;
    }
    out.push({ time: p.time as Time, value: p.value });
    lastTime = p.time;
  }
  return out;
}

/**
 * Stable signature for candle data so we only setData/fitContent when bars change.
 * Includes every bar OHLC — a tip-only signature missed middle-bar updates.
 */
export function candleDataSignature(data: ChartCandle[]): string {
  if (!data.length) return '0';
  return data
    .map((d) => `${d.time},${d.open},${d.high},${d.low},${d.close}`)
    .join(';');
}

/**
 * Stable signature for overlays — include every point so mid-series EMA
 * recalculations trigger setData (tip-only signatures missed interior updates).
 */
export function overlaysSignature(overlays: CandleChartOverlay[]): string {
  if (!overlays.length) return '';
  return overlays
    .map((o) => {
      const title = o.title ?? '';
      const points = o.data
        .map((p) => `${p.time},${p.value}`)
        .join(';');
      return `${o.id}:${o.color}:${title}:${points}`;
    })
    .join('|');
}
