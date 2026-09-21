import { describe, expect, it } from 'vitest';
import { driverNames, leaderCoins, mixLine, shareLabel } from './helpers';

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

  it('formats bid share then to now', () => {
    expect(mixLine({ complete: true, pastBidPct: '55', bidPct: '65', changePct: '+10' })).toBe(
      '55% → 65% (+10)',
    );
    expect(driverNames({ drivers: [{ base: 'ETH', changePct: '+10' }, { base: 'SOL' }] })).toBe(
      'ETH, SOL',
    );
  });
});
