import { describe, expect, it } from 'vitest';
import {
  candleDataSignature,
  chartPriceFormatFromCandles,
  decimalsForMagnitude,
  overlaysSignature,
  toCandlestickData,
  toLineData,
} from './helpers';
import type { ChartCandle } from '@/libs/utils';

function bar(close: number, opts?: Partial<ChartCandle>): ChartCandle {
  return {
    time: opts?.time ?? 1,
    open: opts?.open ?? close,
    high: opts?.high ?? close,
    low: opts?.low ?? close,
    close,
  };
}

describe('CandleChartHost helpers', () => {
  it('maps candles to chart series points', () => {
    expect(
      toCandlestickData([{ time: 100, open: 1, high: 2, low: 0.5, close: 1.5 }]),
    ).toEqual([{ time: 100, open: 1, high: 2, low: 0.5, close: 1.5 }]);
  });

  it('sorts and dedupes line points by time', () => {
    expect(
      toLineData([
        { time: 3, value: 30 },
        { time: 1, value: 10 },
        { time: 2, value: 20 },
        { time: 2, value: 22 },
      ]),
    ).toEqual([
      { time: 1, value: 10 },
      { time: 2, value: 22 },
      { time: 3, value: 30 },
    ]);
  });

  it('drops non-finite line points', () => {
    expect(
      toLineData([
        { time: 1, value: Number.NaN },
        { time: 2, value: 5 },
      ]),
    ).toEqual([{ time: 2, value: 5 }]);
  });

  it('builds stable candle signatures', () => {
    const a = [
      { time: 1, open: 1, high: 1, low: 1, close: 1 },
      { time: 2, open: 2, high: 2, low: 2, close: 2 },
    ];
    const b = [
      { time: 1, open: 1, high: 1, low: 1, close: 1 },
      { time: 2, open: 2, high: 2, low: 2, close: 3 },
    ];
    expect(candleDataSignature(a)).toBe(candleDataSignature(a));
    expect(candleDataSignature(a)).not.toBe(candleDataSignature(b));
    expect(candleDataSignature([])).toBe('0');
  });

  it('builds overlay signatures including empty state', () => {
    expect(overlaysSignature([])).toBe('');
    expect(
      overlaysSignature([
        { id: 'ema-12', color: '#f00', data: [{ time: 1, value: 1 }] },
      ]),
    ).toContain('ema-12');
  });

  it('picks more decimals for smaller magnitudes', () => {
    expect(decimalsForMagnitude(1000)).toBe(2);
    expect(decimalsForMagnitude(12.5)).toBe(4);
    expect(decimalsForMagnitude(0.0000123)).toBeGreaterThanOrEqual(8);
    expect(decimalsForMagnitude(1e-9)).toBe(12);
  });

  it('uses 2dp for large BTC-like prices', () => {
    const fmt = chartPriceFormatFromCandles([
      bar(67000, { open: 66500, high: 68000, low: 66000 }),
    ]);
    expect(fmt.precision).toBe(2);
    expect(fmt.minMove).toBe(0.01);
  });

  it('uses high precision for micro-priced coins so axis is not stuck at 0.00', () => {
    const fmt = chartPriceFormatFromCandles([
      bar(0.00000123, {
        open: 0.0000012,
        high: 0.0000013,
        low: 0.00000115,
      }),
    ]);
    expect(fmt.precision).toBeGreaterThanOrEqual(8);
    expect(fmt.minMove).toBeLessThanOrEqual(1e-8);
    // Crosshair / axis must distinguish these levels (not all 0.00).
    expect(0.00000123 / fmt.minMove).toBeGreaterThan(100);
  });

  it('raises precision when the range is tiny relative to price', () => {
    const fmt = chartPriceFormatFromCandles([
      bar(1.00001, { open: 1.00001, high: 1.00002, low: 1.00001 }),
      bar(1.000015, { time: 2, open: 1.00001, high: 1.00002, low: 1.00001 }),
    ]);
    expect(fmt.precision).toBeGreaterThanOrEqual(5);
  });
});
