import { describe, expect, it } from 'vitest';
import { formatPrice } from './formatPrice';

describe('formatPrice', () => {
  it('returns em dash for empty / non-finite', () => {
    expect(formatPrice(null)).toBe('—');
    expect(formatPrice(undefined)).toBe('—');
    expect(formatPrice('')).toBe('—');
    expect(formatPrice(Number.NaN)).toBe('—');
    expect(formatPrice(Number.POSITIVE_INFINITY)).toBe('—');
  });

  it('formats zero', () => {
    expect(formatPrice(0)).toBe('0');
  });

  it('uses scientific notation for tiny values', () => {
    expect(formatPrice('0.00000012')).toMatch(/e-/i);
  });

  it('uses locale fractions by magnitude tier', () => {
    expect(formatPrice(1234.567)).toMatch(/1[,.]?234/);
    expect(formatPrice(12.34567)).toMatch(/12\.3457|12.3457/);
    expect(formatPrice(0.123456789)).toMatch(/0\.12345679|0.12345679/);
  });

  it('formats negative numbers', () => {
    expect(formatPrice(-0.0000005)).toMatch(/e-/i);
    expect(formatPrice(-42)).toMatch(/-42/);
  });
});
