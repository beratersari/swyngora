import { describe, expect, it } from 'vitest';
import { asksHighToLow, depthPct, maxNotional } from './helpers';

describe('OrderBook helpers', () => {
  it('maxNotional and depthPct', () => {
    const max = maxNotional([
      { notional: '10' },
      { notional: '40' },
      { notional: 'x' },
    ]);
    expect(max).toBe(40);
    expect(depthPct('20', 40)).toBe(50);
    expect(depthPct('0', 40)).toBe(0);
  });

  it('asksHighToLow reverses best-first asks', () => {
    const out = asksHighToLow([{ price: '100.1' }, { price: '100.2' }]);
    expect(out.map((r) => r.price)).toEqual(['100.2', '100.1']);
  });
});
