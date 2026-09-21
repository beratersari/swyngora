import { describe, expect, it } from 'vitest';
import { fearGreedTone, fearGreedWindowLine, recentFearGreed } from './helpers';

describe('fearGreedTone', () => {
  it('maps bands onto up, down, and neutral colors', () => {
    expect(fearGreedTone('extreme_greed')).toBe('success');
    expect(fearGreedTone('greed')).toBe('success');
    expect(fearGreedTone('extreme_fear')).toBe('error');
    expect(fearGreedTone('fear')).toBe('error');
    expect(fearGreedTone('neutral')).toBe('warning');
    expect(fearGreedTone(undefined)).toBe('secondary');
  });
});

describe('fearGreedWindowLine', () => {
  it('prints past to now with a signed change', () => {
    expect(
      fearGreedWindowLine({ window: '24h', pastScore: 48, score: 62, change: 14, complete: true }),
    ).toBe('24h 48 → 62 (+14)');
    expect(fearGreedWindowLine({ window: '1h', complete: false, score: 62 })).toBeNull();
  });
});

describe('recentFearGreed', () => {
  it('keeps the newest bars', () => {
    const rows = Array.from({ length: 60 }, (_, i) => ({ score: i, time: String(i) }));
    const got = recentFearGreed(rows, 48);
    expect(got).toHaveLength(48);
    expect(got[0]?.score).toBe(12);
    expect(got[47]?.score).toBe(59);
  });
});
