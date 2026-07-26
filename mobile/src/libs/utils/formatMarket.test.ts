import { describe, expect, it } from 'vitest';
import { changeTone, formatChangePercent, formatCompactUsd } from './formatMarket';

describe('formatChangePercent', () => {
  it('formats with sign', () => {
    expect(formatChangePercent(1.234)).toBe('+1.23%');
    expect(formatChangePercent(-2.5)).toBe('-2.50%');
  });

  it('dash for empty', () => {
    expect(formatChangePercent(null)).toBe('—');
  });
});

describe('changeTone', () => {
  it('maps sign to tone', () => {
    expect(changeTone(1)).toBe('success');
    expect(changeTone(-1)).toBe('error');
    expect(changeTone(0)).toBe('secondary');
  });
});

describe('formatCompactUsd', () => {
  it('compacts large numbers', () => {
    expect(formatCompactUsd(1_500_000_000)).toBe('1.50B');
    expect(formatCompactUsd(12_300)).toBe('12.30K');
  });

  it('handles infinity marker', () => {
    expect(formatCompactUsd('∞')).toBe('∞');
  });
});
