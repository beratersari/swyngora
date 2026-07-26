import { describe, expect, it } from 'vitest';
import { apiCandlesToChart, resolveInterval, sortedEmaKeys } from './candles';

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
