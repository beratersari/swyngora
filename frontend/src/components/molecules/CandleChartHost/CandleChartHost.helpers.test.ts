import { describe, expect, it } from 'vitest';
import {
  candleDataSignature,
  chartPriceFormatFromCandles,
  decimalsForMagnitude,
  initialVisibleLogicalRange,
  overlaysSignature,
  toCandlestickData,
  toLineData,
} from './CandleChartHost.helpers';
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
  it('right-aligns the first viewport to the latest bars', () => {
    expect(initialVisibleLogicalRange(300, 80, 6)).toEqual({ from: 220, to: 306 });
    expect(initialVisibleLogicalRange(40, 80, 6)).toEqual({ from: 0, to: 46 });
    expect(initialVisibleLogicalRange(0, 80, 6)).toBeNull();
    // 80 real bars + 20 carried: first view stays on the last trade, tail is pannable.
    expect(initialVisibleLogicalRange(100, 80, 6, 80)).toEqual({ from: 0, to: 106 });
  });

  it('maps candles to chart series points', () => {
    expect(
      toCandlestickData([{ time: 100, open: 1, high: 2, low: 0.5, close: 1.5 }]),
    ).toEqual([{ time: 100, open: 1, high: 2, low: 0.5, close: 1.5 }]);
  });

  it('sorts and dedupes candle points by time', () => {
    expect(
      toCandlestickData([
        { time: 300, open: 3, high: 3, low: 3, close: 3 },
        { time: 100, open: 1, high: 1, low: 1, close: 1 },
        { time: 200, open: 2, high: 2, low: 2, close: 2 },
        { time: 200, open: 9, high: 9, low: 9, close: 9 },
      ]),
    ).toEqual([
      { time: 100, open: 1, high: 1, low: 1, close: 1 },
      { time: 200, open: 9, high: 9, low: 9, close: 9 },
      { time: 300, open: 3, high: 3, low: 3, close: 3 },
    ]);
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

describe('toCandlestickData filters non-finite', () => {
  it('drops bars with non-finite OHLC or time', () => {
    expect(
      toCandlestickData([
        { time: 1, open: 1, high: 2, low: 0.5, close: 1.5 },
        { time: 2, open: Number.NaN, high: 2, low: 1, close: 1 },
        { time: Number.NaN, open: 1, high: 1, low: 1, close: 1 },
      ]),
    ).toEqual([{ time: 1, open: 1, high: 2, low: 0.5, close: 1.5 }]);
  });
});

describe('candleDataSignature detects middle-bar changes', () => {
  it('changes when a middle bar OHLC changes with same tips', () => {
    const a = [
      { time: 1, open: 1, high: 1, low: 1, close: 1 },
      { time: 2, open: 2, high: 2, low: 2, close: 2 },
      { time: 3, open: 3, high: 3, low: 3, close: 3 },
    ];
    const b = [
      { time: 1, open: 1, high: 1, low: 1, close: 1 },
      { time: 2, open: 9, high: 9, low: 9, close: 9 },
      { time: 3, open: 3, high: 3, low: 3, close: 3 },
    ];
    expect(candleDataSignature(a)).not.toBe(candleDataSignature(b));
  });
});

describe('overlaysSignature includes title and color', () => {
  it('differs when title changes', () => {
    const base = { id: 'ema-12', color: '#f00', data: [{ time: 1, value: 1 }] };
    expect(overlaysSignature([{ ...base, title: 'A' }])).not.toBe(
      overlaysSignature([{ ...base, title: 'B' }]),
    );
  });

  it('differs when a middle point value changes (same tips and length)', () => {
    const base = {
      id: 'ema-12',
      color: '#0f0',
      title: 'EMA 12',
      data: [
        { time: 1, value: 10 },
        { time: 2, value: 20 },
        { time: 3, value: 30 },
      ],
    };
    const mid = {
      ...base,
      data: [
        { time: 1, value: 10 },
        { time: 2, value: 999 },
        { time: 3, value: 30 },
      ],
    };
    expect(overlaysSignature([base])).not.toBe(overlaysSignature([mid]));
  });
});

describe('chartPriceFormatFromCandles edge cases', () => {
  it('defaults for empty or all-non-finite', () => {
    expect(chartPriceFormatFromCandles([])).toEqual({
      type: 'price',
      precision: 2,
      minMove: 0.01,
    });
    expect(
      chartPriceFormatFromCandles([
        { time: 1, open: Number.NaN, high: Number.NaN, low: Number.NaN, close: Number.NaN },
      ]),
    ).toEqual({ type: 'price', precision: 2, minMove: 0.01 });
  });
});

describe('decimalsForMagnitude edges', () => {
  it('handles zero and non-finite', () => {
    expect(decimalsForMagnitude(0)).toBe(2);
    expect(decimalsForMagnitude(Number.NaN)).toBe(2);
    expect(decimalsForMagnitude(0.05)).toBe(6);
    expect(decimalsForMagnitude(0.5)).toBe(5);
  });
});
