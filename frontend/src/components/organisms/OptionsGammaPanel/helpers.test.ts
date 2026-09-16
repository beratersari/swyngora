import { describe, expect, it } from 'vitest';
import { formatFlipDist, formatGex, gexBarPct, gammaHasData, isNearFlip, maxAbsGex } from './helpers';

describe('options gamma helpers', () => {
  it('formats signed compact GEX', () => {
    expect(formatGex('1250000000')).toBe('+1.25B');
    expect(formatGex('-840000000')).toBe('−840.00M');
    expect(formatGex(undefined)).toBe('—');
  });

  it('formats flip distance', () => {
    expect(formatFlipDist('1.25')).toBe('+1.25%');
    expect(formatFlipDist('-0.4')).toBe('−0.40%');
    expect(formatFlipDist('0')).toBeUndefined();
  });

  it('scales profile bars', () => {
    expect(maxAbsGex([{ spot: '1', netUsd: '-10' }, { spot: '2', netUsd: '40' }])).toBe(40);
    expect(gexBarPct('20', 40)).toBe(50);
  });

  it('marks the flip column', () => {
    expect(isNearFlip('100200', '100000')).toBe(true);
    expect(isNearFlip('110000', '100000')).toBe(false);
    expect(gammaHasData({ netUsd: '-1' })).toBe(true);
    expect(gammaHasData(undefined)).toBe(false);
  });
});
