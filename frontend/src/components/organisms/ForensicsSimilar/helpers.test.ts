import { describe, expect, it } from 'vitest';
import { orderedSimilarFields, similarAsOfLabel, similarFieldScore, similarMoveAt } from './helpers';

describe('ForensicsSimilar helpers', () => {
  it('orders compared fields and labels missing as n/a', () => {
    const rows = orderedSimilarFields([
      { name: 'oi', used: true, score: '88' },
      { name: 'price', used: true, score: '70' },
      { name: 'book', used: false },
    ]);
    expect(rows.map((r) => r.name)).toEqual(['price', 'oi', 'book']);
    expect(similarFieldScore(rows[2]!)).toBe('n/a');
    expect(similarFieldScore(rows[0]!)).toBe('70');
  });

  it('names a frozen comparison clock and stays quiet for a live one', () => {
    const now = Date.parse('2026-08-20T16:00:00.000Z');
    expect(similarAsOfLabel('2026-08-20T14:15:00.000Z', now)).toBe('2026-08-20 14:15:00 UTC');
    expect(similarAsOfLabel('2026-08-20T15:59:30.000Z', now)).toBeNull();
    expect(similarAsOfLabel(undefined, now)).toBeNull();
  });

  it('normalizes a similar-move event time', () => {
    expect(similarMoveAt({ at: '2026-08-18T12:00:00Z' })).toBe('2026-08-18T12:00:00.000Z');
    expect(similarMoveAt({ at: 'not-a-time' })).toBeNull();
    expect(similarMoveAt({})).toBeNull();
  });
});
