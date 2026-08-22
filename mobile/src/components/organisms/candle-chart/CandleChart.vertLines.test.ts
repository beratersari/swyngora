import { snapVertLinesToCandleTimes, isoToUnixSeconds } from './CandleChart.vertLines';

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

  it('snaps between bars and clamps outside the window', () => {
    expect(snapVertLinesToCandleTimes([{ id: 'mid', time: 240 }], candles)).toEqual([
      { id: 'mid', time: 200 },
    ]);
    expect(snapVertLinesToCandleTimes([{ id: 'future', time: 9_999 }], candles)).toEqual([
      { id: 'future', time: 300 },
    ]);
  });

  it('returns empty without candles', () => {
    expect(snapVertLinesToCandleTimes([{ id: 'a', time: 200 }], [])).toEqual([]);
  });
});
