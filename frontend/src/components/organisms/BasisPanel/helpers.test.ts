import { describe, expect, it } from 'vitest';
import { kindTone, richerLine, shapeTone, slopeLabel, venueLabel } from './helpers';

describe('BasisPanel helpers', () => {
  it('maps kind, shape, and richer', () => {
    expect(kindTone('premium')).toBe('premium');
    expect(kindTone('nope')).toBe('flat');
    expect(shapeTone('backwardation')).toBe('backwardation');
    expect(shapeTone(undefined)).toBe('flat');
    expect(richerLine('far')).toBe('richerFar');
    expect(richerLine('near')).toBe('richerNear');
    expect(venueLabel('binance')).toBe('Binance');
    expect(slopeLabel('up')).toBe('up');
  });
});
