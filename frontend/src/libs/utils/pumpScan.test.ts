import { describe, expect, it } from 'vitest';
import { pumpScanHitToRow, pumpScanHitsToRows } from './pumpScan';

/** Fixture shaped like live GET /api/v1/market/pumps/scan hits. */
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
      closeTime: '2026-07-27T06:29:59.999Z',
      returnPct: 22.811983958480763,
      volumeRatio: 17.5664348678538,
      mode: 'close_return',
    },
    {
      index: 67,
      openTime: '2026-07-27T08:30:00Z',
      returnPct: 9.31,
      volumeRatio: 3.5,
    },
  ],
};

describe('pumpScan', () => {
  it('reads return/vol/time from nested best event, not hit root', () => {
    const row = pumpScanHitToRow(liveHit);
    expect(row).not.toBeNull();
    expect(row!.symbol).toBe('BANKUSDT');
    expect(row!.returnPct).toBeCloseTo(22.81, 1);
    expect(row!.volumeRatio).toBeCloseTo(17.56, 1);
    expect(row!.openTime).toBe('2026-07-27T06:15:00Z');
    expect(row!.eventCount).toBe(2);
    expect(row!.interval).toBe('15m');
  });

  it('uses API best-event fields on the hit root', () => {
    const row = pumpScanHitToRow({
      symbol: 'X',
      bestReturnPct: 5,
      bestVolumeRatio: 2,
      bestOpenTime: '2024-01-01T00:00:00Z',
    });
    expect(row!.returnPct).toBe(5);
    expect(row!.volumeRatio).toBe(2);
    expect(row!.openTime).toBe('2024-01-01T00:00:00Z');
  });

  it('returns null without symbol', () => {
    expect(pumpScanHitToRow({ bestReturnPct: 1 })).toBeNull();
  });

  it('maps list and skips empty symbols', () => {
    expect(pumpScanHitsToRows([liveHit, { symbol: '' }])).toHaveLength(1);
    expect(pumpScanHitsToRows(undefined)).toEqual([]);
  });

});
