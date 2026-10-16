import { describe, expect, it } from 'vitest';
import { correlationKind, swingChange } from './helpers';

describe('correlationKind', () => {
  it('says the risk is not spread when the coins move together', () => {
    expect(
      correlationKind({
        holdings: [
          { asset: 'BTC', notional: 100, side: 'long' },
          { asset: 'ETH', notional: 100, side: 'long' },
        ],
        together: 0.9,
        swingPct: 4,
        apartPct: 2,
        pairs: [],
        missing: [],
      }),
    ).toBe('together');
  });

  it('says a short offsets a shared move', () => {
    expect(
      correlationKind({
        holdings: [
          { asset: 'BTC', notional: 100, side: 'long' },
          { asset: 'ETH', notional: -100, side: 'short' },
        ],
        together: 0.9,
        swingPct: 0.2,
        apartPct: 2,
        pairs: [],
        missing: [],
      }),
    ).toBe('hedge');
  });

  it('does not invent a pair for one coin', () => {
    expect(
      correlationKind({
        holdings: [{ asset: 'BTC', notional: 100, side: 'long' }],
        swingPct: 3,
        apartPct: 3,
        pairs: [],
        missing: [],
      }),
    ).toBe('one');
  });
});

describe('swingChange', () => {
  it('raises, lowers, or stays quiet', () => {
    expect(swingChange(2, 4, true)).toBe('up');
    expect(swingChange(4, 2, true)).toBe('down');
    expect(swingChange(2, 2.01, true)).toBe('same');
    expect(swingChange(2, 3, false)).toBe('unknown');
    expect(swingChange(undefined, 3, true)).toBe('first');
  });
});
