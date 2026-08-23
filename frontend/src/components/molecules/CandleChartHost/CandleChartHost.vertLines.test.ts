import { describe, expect, it } from 'vitest';
import {
  isoToUnixSeconds,
  snapVertLinesToCandleTimes,
} from './CandleChartHost.vertLines';

describe('isoToUnixSeconds', () => {
  it('parses RFC3339', () => {
    expect(isoToUnixSeconds('2026-09-03T00:00:00Z')).toBe(
      Math.floor(Date.parse('2026-09-03T00:00:00Z') / 1000),
    );
  });

  it('returns null for empty or invalid', () => {
    expect(isoToUnixSeconds(null)).toBeNull();
    expect(isoToUnixSeconds('')).toBeNull();
    expect(isoToUnixSeconds('nope')).toBeNull();
  });
});

describe('snapVertLinesToCandleTimes', () => {
  const candles = [100, 200, 300];

  it('keeps an exact bar time', () => {
    expect(snapVertLinesToCandleTimes([{ id: 'a', time: 200 }], candles)).toEqual([
      { id: 'a', time: 200 },
    ]);
  });

  it('snaps to the bar that contains the event, not the next open', () => {
    expect(snapVertLinesToCandleTimes([{ id: 'a', time: 240 }], candles)).toEqual([
      { id: 'a', time: 200 },
    ]);
    expect(snapVertLinesToCandleTimes([{ id: 'a', time: 260 }], candles)).toEqual([
      { id: 'a', time: 200 },
    ]);
  });

  it('drops events before the first bar or after the last bar closes', () => {
    expect(snapVertLinesToCandleTimes([{ id: 'past', time: 10 }], candles, { barDurationSec: 100 })).toEqual(
      [],
    );
    expect(
      snapVertLinesToCandleTimes([{ id: 'future', time: 9_999 }], candles, { barDurationSec: 100 }),
    ).toEqual([]);
  });

  it('keeps a halt in the last forming bar', () => {
    expect(
      snapVertLinesToCandleTimes([{ id: 'halt', time: 350 }], candles, { barDurationSec: 100 }),
    ).toEqual([{ id: 'halt', time: 300 }]);
  });

  it('returns empty when there are no candles or no valid times', () => {
    expect(snapVertLinesToCandleTimes([{ id: 'a', time: 200 }], [])).toEqual([]);
    expect(snapVertLinesToCandleTimes([{ id: 'a', time: 0 }], candles)).toEqual([]);
  });

  it.each([
    { interval: '1m', bar: 60, haltHour: 20, haltMin: 40 },
    { interval: '5m', bar: 300, haltHour: 20, haltMin: 43 },
    { interval: '15m', bar: 900, haltHour: 20, haltMin: 44 },
    { interval: '1h', bar: 3600, haltHour: 20, haltMin: 50 },
    { interval: '4h', bar: 14400, haltHour: 22, haltMin: 0 },
    { interval: '1d', bar: 86400, haltHour: 20, haltMin: 0 },
    { interval: '1w', bar: 604800, haltHour: 20, haltMin: 0 },
    { interval: '1M', bar: 30 * 86400, haltHour: 20, haltMin: 0 },
  ])('places a late-session halt on the $interval candle that contains it', ({
    bar,
    haltHour,
    haltMin,
  }) => {
    const day0 = Date.parse('2026-08-17T00:00:00Z') / 1000;
    const opens: number[] = [];
    for (let t = day0; t < day0 + 3 * 86400; t += bar) opens.push(t);
    const halt = Date.parse(
      `2026-08-17T${String(haltHour).padStart(2, '0')}:${String(haltMin).padStart(2, '0')}:00Z`,
    ) / 1000;
    const want = opens.reduce((acc, open) => (open <= halt ? open : acc), opens[0]!);
    const got = snapVertLinesToCandleTimes([{ id: 'delist-halt', time: halt }], opens, {
      barDurationSec: bar,
    });
    expect(got).toEqual([{ id: 'delist-halt', time: want }]);
    const next = want + bar;
    expect(got[0]?.time).not.toBe(next);
  });
});
