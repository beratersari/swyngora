import { describe, expect, it } from 'vitest';
import {
  apiCandlesToChart,
  filterValidApiCandles,
  mergeCandleHistory,
  oldestCandleOpenTimeMs,
  trimCandlesToMax,
  type ApiCandle,
} from './candles';

function c(openTime: string, close = '1'): ApiCandle {
  return {
    openTime,
    open: close,
    high: close,
    low: close,
    close,
    volume: '0',
  };
}

describe('apiCandlesToChart', () => {
  it('maps valid candles to chart points with second timestamps', () => {
    const out = apiCandlesToChart([
      {
        openTime: '2024-01-01T00:00:00.000Z',
        open: '1',
        high: '2',
        low: '0.5',
        close: '1.5',
        volume: '10',
      },
    ]);
    expect(out).toHaveLength(1);
    expect(out[0]?.time).toBe(Math.floor(Date.parse('2024-01-01T00:00:00.000Z') / 1000));
    expect(out[0]?.close).toBe(1.5);
  });

  it('drops invalid openTime or non-numeric OHLC', () => {
    expect(
      apiCandlesToChart([
        { openTime: 'bad', open: '1', high: '1', low: '1', close: '1', volume: '0' },
        {
          openTime: '2024-01-01T00:00:00Z',
          open: 'x',
          high: '1',
          low: '1',
          close: '1',
          volume: '0',
        },
      ]),
    ).toEqual([]);
  });

  it('handles empty and mixed valid/invalid', () => {
    expect(apiCandlesToChart([])).toEqual([]);
    const mixed = apiCandlesToChart([
      { openTime: 'bad', open: '1', high: '1', low: '1', close: '1', volume: '0' },
      {
        openTime: '2024-01-01T00:00:00Z',
        open: '1',
        high: '2',
        low: '0.5',
        close: '1.5',
        volume: '1',
      },
    ]);
    expect(mixed).toHaveLength(1);
  });
});

describe('mergeCandleHistory', () => {
  it('merges by openTime, newer wins, sorted ascending', () => {
    const older = [c('2024-01-01T00:00:00Z', '1'), c('2024-01-01T01:00:00Z', '2')];
    const newer = [c('2024-01-01T01:00:00Z', '9'), c('2024-01-01T02:00:00Z', '3')];
    const out = mergeCandleHistory(older, newer);
    expect(out.map((x) => x.close)).toEqual(['1', '9', '3']);
  });
});

describe('oldestCandleOpenTimeMs', () => {
  it('returns earliest openTime ms', () => {
    const ms = oldestCandleOpenTimeMs([
      c('2024-01-02T00:00:00Z'),
      c('2024-01-01T00:00:00Z'),
    ]);
    expect(ms).toBe(Date.parse('2024-01-01T00:00:00Z'));
  });

  it('returns null for empty', () => {
    expect(oldestCandleOpenTimeMs([])).toBeNull();
  });
});

describe('trimCandlesToMax', () => {
  it('keeps newest bars when over max', () => {
    const bars = [
      c('2024-01-01T00:00:00Z'),
      c('2024-01-01T01:00:00Z'),
      c('2024-01-01T02:00:00Z'),
    ];
    expect(trimCandlesToMax(bars, 2).map((x) => x.openTime)).toEqual([
      '2024-01-01T01:00:00Z',
      '2024-01-01T02:00:00Z',
    ]);
  });
});

describe('filterValidApiCandles', () => {
  it('drops incomplete rows', () => {
    expect(
      filterValidApiCandles([
        { openTime: '2024-01-01T00:00:00Z', open: '1', high: '1', low: '1', close: '1' },
        { openTime: 'bad', open: '1', high: '1', low: '1', close: '1' },
        undefined,
      ]),
    ).toHaveLength(1);
  });
});
