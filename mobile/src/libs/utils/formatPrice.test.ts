import { describe, expect, it } from 'vitest';
import { formatPrice } from './formatPrice';

describe('formatPrice', () => {
  it('returns em dash for empty', () => {
    expect(formatPrice(null)).toBe('—');
    expect(formatPrice(undefined)).toBe('—');
    expect(formatPrice('')).toBe('—');
  });

  it('formats zero', () => {
    expect(formatPrice(0)).toBe('0');
  });

  it('uses scientific notation for tiny values', () => {
    expect(formatPrice('0.00000012')).toMatch(/e-/i);
  });
});
