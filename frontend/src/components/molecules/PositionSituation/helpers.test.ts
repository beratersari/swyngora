import { describe, expect, it } from 'vitest';
import { openMarginFor } from './helpers';

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
