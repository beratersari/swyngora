import { describe, expect, it } from 'vitest';
import { leaderCoins, shareLabel } from './helpers';

describe('PerpLiquidityOverview helpers', () => {
  it('drops empty or errored coins then keeps the largest', () => {
    const got = leaderCoins({
      coins: [
        { symbol: 'SOLUSDT', error: 'no perpetual depth' },
        { symbol: 'BTCUSDT', base: 'BTC', total: '100', sharePct: '80' },
        { symbol: 'ETHUSDT', base: 'ETH', total: '20', sharePct: '20' },
      ],
    });
    expect(got.map((c) => c.base)).toEqual(['BTC', 'ETH']);
  });

  it('formats a share percent', () => {
    expect(shareLabel('48')).toBe('48%');
    expect(shareLabel(undefined)).toBe('—');
  });
});
