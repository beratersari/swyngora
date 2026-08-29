import { describe, expect, it } from 'vitest';
import { pumpScanHitToRow, pumpScanHitsToRows } from './PumpsPage.helpers';

const liveHit = {
  symbol: 'BANKUSDT',
  exchange: 'binance',
  interval: '15m',
  bestReturnPct: 22.811983958480763,
  bestVolumeRatio: 17.5664348678538,
  bestOpenTime: '2026-07-27T06:15:00Z',
  events: [
    {
      index: 58,
      openTime: '2026-07-27T06:15:00Z',
      returnPct: 22.811983958480763,
      volumeRatio: 17.5664348678538,
    },
  ],
};

describe('pump scan mapping', () => {
  it('maps API best-event fields', () => {
    const row = pumpScanHitToRow(liveHit);
    expect(row).not.toBeNull();
    expect(row!.returnPct).toBeCloseTo(22.81, 1);
    expect(row!.volumeRatio).toBeCloseTo(17.56, 1);
    expect(row!.openTime).toBe('2026-07-27T06:15:00Z');
  });

  it('maps list and skips empty symbols', () => {
    expect(pumpScanHitsToRows([liveHit, { symbol: '' }])).toHaveLength(1);
  });
});
