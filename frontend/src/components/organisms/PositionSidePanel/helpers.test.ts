import { describe, expect, it } from 'vitest';
import { sideLine, signedTapeAmount } from './helpers';

describe('signedTapeAmount', () => {
  it('keeps the sign on compact amounts', () => {
    expect(signedTapeAmount('45000000')).toBe('+45.00M');
    expect(signedTapeAmount('-15000000')).toBe('-15.00M');
    expect(signedTapeAmount('0')).toBe('0');
  });
});

describe('sideLine', () => {
  it('names both sides', () => {
    expect(sideLine('45000000', '-15000000')).toBe('Long +45.00M · Short -15.00M');
  });
});
