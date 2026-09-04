import { describe, expect, it } from 'vitest';
import { setupRadarAlertHref } from './helpers';

describe('ForensicsNowScan helpers', () => {
  it('builds a Setup Radar alert link', () => {
    expect(setupRadarAlertHref('BTCUSDT', 'binance')).toBe(
      '/alerts?kind=setup_radar&symbol=BTCUSDT&exchange=binance',
    );
    expect(setupRadarAlertHref('ALL')).toBe('/alerts?kind=setup_radar&symbol=ALL');
  });
});
