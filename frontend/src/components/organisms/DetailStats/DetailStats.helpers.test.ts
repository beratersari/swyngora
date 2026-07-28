import { describe, expect, it } from 'vitest';
import { formatMaxSupply } from './DetailStats.helpers';

const fmt = (v: number | null | undefined) =>
  v === null || v === undefined ? '—' : String(v);

describe('formatMaxSupply', () => {
  it('shows dash when supply snapshot is missing (loading / failed)', () => {
    expect(formatMaxSupply(undefined, '∞ / n/a', fmt)).toBe('—');
    expect(formatMaxSupply(null, '∞ / n/a', fmt)).toBe('—');
  });

  it('shows open label only when snapshot exists and max is nullish', () => {
    expect(formatMaxSupply({ maxSupply: null }, '∞ / n/a', fmt)).toBe('∞ / n/a');
    expect(formatMaxSupply({ maxSupply: undefined }, '∞ / n/a', fmt)).toBe('∞ / n/a');
    expect(formatMaxSupply({}, '∞ / n/a', fmt)).toBe('∞ / n/a');
  });

  it('formats a finite max supply', () => {
    expect(formatMaxSupply({ maxSupply: 21_000_000 }, '∞ / n/a', fmt)).toBe('21000000');
  });
});
