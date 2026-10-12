import { describe, expect, it } from 'vitest';
import { situationCandleCloseMs } from './helpers';

describe('situationCandleCloseMs', () => {
  it('uses the candle close when that close is already in the past', () => {
    const open = 1_700_000_000;
    const bar = 3600;
    const now = (open + 7200) * 1000;
    expect(situationCandleCloseMs(open, bar, now)).toBe((open + bar) * 1000);
  });

  it('keeps a close that lands on now', () => {
    const open = 1_000;
    const bar = 60;
    const now = (open + bar) * 1000;
    expect(situationCandleCloseMs(open, bar, now)).toBe(now);
  });

  it('holds a forming candle on the current minute', () => {
    const now = Date.UTC(2026, 5, 1, 16, 7, 42);
    const open = Math.floor(now / 1000) - 120;
    expect(situationCandleCloseMs(open, 3600, now)).toBe(Math.floor(now / 60_000) * 60_000);
  });

  it('rejects a bar that has no length', () => {
    expect(situationCandleCloseMs(1, 0, 1)).toBeNull();
    expect(situationCandleCloseMs(Number.NaN, 60, 1)).toBeNull();
  });
});
