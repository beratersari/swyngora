import { describe, expect, it } from 'vitest';
import { alertDisplayLabel, parseFiniteNumber } from './alerts';

describe('parseFiniteNumber', () => {
  it('parses and rejects', () => {
    expect(parseFiniteNumber('12.5')).toBe(12.5);
    expect(parseFiniteNumber(null)).toBeNull();
    expect(parseFiniteNumber('x')).toBeNull();
  });
});

describe('alertDisplayLabel', () => {
  it('formats kinds', () => {
    expect(alertDisplayLabel({ symbol: 'BTCUSDT', kind: 'price_above', threshold: 1 })).toContain(
      '≥',
    );
  });
});
