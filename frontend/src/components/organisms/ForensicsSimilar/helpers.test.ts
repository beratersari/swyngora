import { describe, expect, it } from 'vitest';
import { orderedSimilarFields, similarFieldScore } from './helpers';

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
});
