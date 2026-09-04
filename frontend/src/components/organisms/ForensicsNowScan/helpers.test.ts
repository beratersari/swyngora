import { describe, expect, it } from 'vitest';
import { orderedScanFields, scanFieldScore, setupRadarAlertHref } from './helpers';

describe('ForensicsNowScan helpers', () => {
  it('builds a Setup Radar alert link', () => {
    expect(setupRadarAlertHref('BTCUSDT', 'binance')).toBe(
      '/alerts?kind=setup_radar&symbol=BTCUSDT&exchange=binance',
    );
    expect(setupRadarAlertHref('ALL')).toBe('/alerts?kind=setup_radar&symbol=ALL');
  });

  it('orders tape fields and labels missing scores', () => {
    expect(
      orderedScanFields([
        { name: 'oi', used: true, score: '91' },
        { name: 'price', used: true, score: '72' },
        { name: 'takers', used: false },
      ]).map((f) => f.name),
    ).toEqual(['price', 'takers', 'oi']);
    expect(scanFieldScore({ name: 'takers', used: false })).toBe('n/a');
    expect(scanFieldScore({ name: 'price', used: true, score: '72' })).toBe('72');
  });
});
