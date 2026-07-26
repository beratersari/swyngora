import { describe, expect, it } from 'vitest';
import { apiCandlesToChart } from './candles';

describe('apiCandlesToChart', () => {
  it('maps valid candles to chart bars', () => {
    const out = apiCandlesToChart([
      {
        openTime: '2024-01-01T00:00:00.000Z',
        open: '100',
        high: '110',
        low: '90',
        close: '105',
        volume: '1',
      },
    ]);
    expect(out).toHaveLength(1);
    expect(out[0].open).toBe(100);
    expect(out[0].close).toBe(105);
    expect(out[0].time).toBe(Math.floor(Date.parse('2024-01-01T00:00:00.000Z') / 1000));
  });

  it('drops invalid rows', () => {
    expect(
      apiCandlesToChart([
        {
          openTime: 'bad',
          open: 'x',
          high: '1',
          low: '1',
          close: '1',
          volume: '1',
        },
      ]),
    ).toHaveLength(0);
  });
});
