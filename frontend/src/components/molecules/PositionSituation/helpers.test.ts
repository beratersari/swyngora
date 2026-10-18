import { describe, expect, it } from 'vitest';
import { openMarginFor, storedReadComparison } from './helpers';

describe('openMarginFor', () => {
  it('keeps the open rows for this coin', () => {
    const rows = openMarginFor(
      [
        { exchange: 'binance', symbol: 'BTCUSDT', side: 'long', status: 'open', quantity: 1, leverage: 5 },
        { exchange: 'binance', symbol: 'BTCUSDT', side: 'short', status: 'closed', quantity: 1 },
        { exchange: 'bybit', symbol: 'BTCUSDT', side: 'long', status: 'open', quantity: 1 },
        { exchange: 'binance', symbol: 'ETHUSDT', side: 'long', status: 'open', quantity: 1 },
      ],
      'binance',
      'BTCUSDT',
    );
    expect(rows).toHaveLength(1);
    expect(rows[0]?.side).toBe('long');
  });
});

describe('storedReadComparison', () => {
  const entry = { priceCvd: { kind: 'price_up_cvd_down' as const } };

  it('names a kind that left and stays quiet when the live read failed', () => {
    const cmp = storedReadComparison(entry, { priceCvd: { kind: 'agrees' } }, true);
    expect(cmp?.same).toBe(false);
    expect(cmp?.clauses?.[0]).toMatchObject({ clause: 'priceCvd', status: 'changed', live: 'agrees' });
    expect(storedReadComparison(entry, undefined, false)).toBeUndefined();
    expect(storedReadComparison(undefined, { priceCvd: { kind: 'agrees' } }, true)).toBeUndefined();
  });
});
