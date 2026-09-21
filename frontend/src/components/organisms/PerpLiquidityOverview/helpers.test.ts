import { describe, expect, it } from 'vitest';
import { bandLeaderLabel, bandShareLabel, distanceBands, driverNames, leaderCoins, mixLine, shareLabel } from './helpers';

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

  it('labels how much of the 1% book sits inside a tighter band', () => {
    const bands = distanceBands({
      rangePct: '1',
      bands: [
        { rangePct: '0.1', total: '80000000', ofHeadlinePct: '8', leaders: ['ETH', 'SOL'] },
        { rangePct: '1', total: '1000000000', ofHeadlinePct: '100', leaders: ['BTC'] },
      ],
    });
    expect(bandShareLabel(bands[0], '1')).toBe('80.00M · 8%');
    expect(bandShareLabel(bands[1], '1')).toBe('1.00B');
    expect(bandLeaderLabel(bands[0])).toBe('ETH, SOL');
  });
});
