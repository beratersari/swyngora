import { describe, expect, it } from 'vitest';
import {
  apiCandlesToChart,
  endTimeBeforeOldestCandle,
  mergeChartCandles,
  resolveInterval,
  sortedEmaKeys,
  type ChartCandle,
} from './candles';

describe('apiCandlesToChart', () => {
  it('maps valid candles', () => {
    const out = apiCandlesToChart([
      {
        openTime: '2026-01-01T00:00:00.000Z',
        open: '1',
        high: '2',
        low: '0.5',
        close: '1.5',
      },
    ]);
    expect(out).toHaveLength(1);
    expect(out[0].open).toBe(1);
    expect(out[0].time).toBe(Date.parse('2026-01-01T00:00:00.000Z') / 1000);
  });

  it('drops invalid', () => {
    expect(apiCandlesToChart([{ openTime: 'x', open: 'a', high: '1', low: '1', close: '1' }])).toEqual(
      [],
    );
  });
});

describe('resolveInterval', () => {
  it('prefers supported preferred', () => {
    expect(resolveInterval('4h', ['1h', '4h', '1d'])).toBe('4h');
  });
  it('falls back', () => {
    expect(resolveInterval('2h', ['1h', '4h'])).toBe('1h');
  });
});

describe('sortedEmaKeys', () => {
  it('sorts numeric keys', () => {
    expect(sortedEmaKeys({ '26': 1, '12': 2 })).toEqual(['12', '26']);
  });
});

describe('mergeChartCandles', () => {
  const a: ChartCandle = { time: 100, open: 1, high: 2, low: 1, close: 1.5 };
  const b: ChartCandle = { time: 200, open: 2, high: 3, low: 2, close: 2.5 };
  const bUpdated: ChartCandle = { time: 200, open: 2, high: 4, low: 2, close: 3 };

  it('merges and sorts by time', () => {
    expect(mergeChartCandles([b], [a])).toEqual([a, b]);
  });

  it('later series wins on same time', () => {
    expect(mergeChartCandles([b], [bUpdated])[0].high).toBe(4);
  });
});

describe('endTimeBeforeOldestCandle', () => {
  it('returns RFC3339 just before open', () => {
    const iso = endTimeBeforeOldestCandle({
      time: 1_700_000_000,
      open: 1,
      high: 1,
      low: 1,
      close: 1,
    });
    expect(iso).toBe(new Date(1_700_000_000 * 1000 - 1).toISOString());
    expect(endTimeBeforeOldestCandle(undefined)).toBeUndefined();
  });
});
