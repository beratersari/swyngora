import { describe, expect, it } from 'vitest';
import {
  bookHeatColumns,
  crowdTopLines,
  footprintLevels,
  footprintWindow,
  formatRecordedSpan,
  futuresRecorded,
  liquidationProfile,
} from './chartDesk';

describe('chart desk mappers', () => {
  it('picks a flow window from the visible span', () => {
    expect(footprintWindow(4 * 60)).toBe('5m');
    expect(footprintWindow(50 * 60)).toBe('1h');
    expect(footprintWindow(5 * 3600)).toBe('1d');
    expect(footprintWindow(10 * 86400)).toBe('7d');
  });

  it('keeps the largest book levels and thins columns', () => {
    const columns = Array.from({ length: 180 }, (_, i) => ({
      t: new Date(Date.UTC(2026, 8, 29, 12, 0, i)).toISOString(),
      bids: [
        { price: '100', notional: '1' },
        { price: '99', notional: '50' },
      ],
      asks: [{ price: '101', notional: '20' }],
    }));
    const cols = bookHeatColumns({ columns });
    expect(cols.length).toBeLessThanOrEqual(90);
    expect(cols[0]?.bids[0]?.price).toBe(99);
    expect(cols[0]?.asks[0]?.notional).toBe(20);
  });

  it('reads combined footprint rows', () => {
    const levels = footprintLevels({
      combined: {
        levels: [
          { price: '100', buyVolume: '8', sellVolume: '2', volume: '10' },
          { price: '101', buyVolume: '0', sellVolume: '0', volume: '0' },
        ],
      },
    });
    expect(levels).toEqual([{ price: 100, buy: 8, sell: 2 }]);
  });

  it('keeps the largest liquidation pockets on each side', () => {
    const bars = liquidationProfile({
      kind: 'levels',
      levels: [
        { price: '90', longNotional: '5', shortNotional: '0' },
        { price: '110', longNotional: '0', shortNotional: '9' },
      ],
    });
    expect(bars).toEqual([
      { price: 90, notional: 5, side: 'long' },
      { price: 110, notional: 9, side: 'short' },
    ]);
  });

  it('plots crowd and top-trader long percent', () => {
    const lines = crowdTopLines({
      gapHistory: [{ time: '2026-09-29T12:00:00Z', crowdLongPct: '48', topLongPct: '61' }],
    });
    expect(lines.crowd).toHaveLength(1);
    expect(lines.top[0]?.value).toBe(61);
  });

  it('formats futures recording spans', () => {
    expect(formatRecordedSpan(90)).toBe('2m');
    expect(formatRecordedSpan(3 * 3600)).toBe('3h');
    expect(formatRecordedSpan(3 * 86400 + 2 * 3600)).toBe('3d 2h');
    expect(
      futuresRecorded({
        series: [
          { kind: 'futures_oi', exchange: 'binance', spanSec: 3600 },
          { kind: 'book', exchange: 'bybit', spanSec: 10 },
        ],
      }),
    ).toEqual([{ exchange: 'binance', spanSec: 3600 }]);
  });
});
