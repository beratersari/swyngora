import { describe, expect, it } from 'vitest';
import { apiCandlesToChart } from './candles';

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
