import { describe, expect, it } from 'vitest';
import {
  episodeLeadBeats,
  formatClock,
  formatDurationSec,
  formatRatio,
  formatSharePct,
  gradeTone,
  orderedWindows,
  sideTone,
} from './helpers';

describe('LiquidationCascade helpers', () => {
  it('formats ratios and grades', () => {
    expect(formatRatio(4.2)).toBe('4.2×');
    expect(formatRatio(12)).toBe('12×');
    expect(formatRatio(0)).toBe('—');
    expect(gradeTone('cascade')).toBe('cascade');
    expect(gradeTone('nope')).toBe('quiet');
    expect(sideTone('long')).toBe('long');
    expect(sideTone(undefined)).toBe('none');
  });

  it('fills missing burst windows', () => {
    const rows = orderedWindows([{ window: '5m', maxRatio: 3 }]);
    expect(rows.map((r) => r.window)).toEqual(['1m', '5m', '15m']);
    expect(rows[1]?.maxRatio).toBe(3);
  });

  it('formats duration and UTC clocks', () => {
    expect(formatDurationSec(45)).toBe('45s');
    expect(formatDurationSec(600)).toBe('10m');
    expect(formatDurationSec(3600)).toBe('1h');
    expect(formatDurationSec(5400)).toBe('1h 30m');
    expect(formatClock('2026-08-30T15:21:00.000Z')).toBe('15:21:00');
    expect(formatClock('2026-08-30T15:21:17.000Z')).toBe('15:21:17');
  });

  it('orders combined venue lead and follow seconds', () => {
    expect(
      episodeLeadBeats({
        combined: true,
        largestBy: 'bybit',
        venueStarts: [
          { exchange: 'binance', lagSec: 0, totalNotional: '20000', sharePct: 20 },
          { exchange: 'bybit', lagSec: 12, totalNotional: '80000', sharePct: 80 },
          { exchange: 'okx', lagSec: 40, totalNotional: '0', sharePct: 0 },
        ],
      }),
    ).toEqual([
      { exchange: 'binance', lagSec: 0, role: 'first', totalNotional: '20000', sharePct: 20, largest: false },
      { exchange: 'bybit', lagSec: 12, role: 'follow', totalNotional: '80000', sharePct: 80, largest: true },
      { exchange: 'okx', lagSec: 40, role: 'follow', totalNotional: '0', sharePct: 0, largest: false },
    ]);
    expect(
      episodeLeadBeats({
        combined: true,
        orderUnknown: true,
        venueStarts: [
          { exchange: 'binance', lagSec: 0 },
          { exchange: 'bybit', lagSec: 0 },
        ],
      }).map((b) => b.role),
    ).toEqual(['tied', 'tied']);
    expect(formatSharePct(80)).toBe('80%');
    expect(formatSharePct(12.4)).toBe('12.4%');
  });
});
