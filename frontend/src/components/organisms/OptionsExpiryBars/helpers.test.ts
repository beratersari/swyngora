import { describe, expect, it } from 'vitest';
import { callShare, expiryShare } from './helpers';

describe('options expiry helpers', () => {
  it('clamps share percent', () => {
    expect(expiryShare({ sharePct: '25' })).toBe(25);
    expect(expiryShare({ sharePct: '150' })).toBe(100);
  });

  it('splits call vs put', () => {
    expect(callShare({ callOi: '75', putOi: '25' })).toBe(0.75);
    expect(callShare({})).toBe(0.5);
  });
});
