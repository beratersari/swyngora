import { describe, expect, it } from 'vitest';
import { alertDisplayLabel, evaluateAlert, parseFiniteNumber } from './alerts';

describe('evaluateAlert', () => {
  it('price above/below', () => {
    expect(evaluateAlert('price_above', 100, 100, 0)).toBe(true);
    expect(evaluateAlert('price_above', 100, 99, 0)).toBe(false);
    expect(evaluateAlert('price_below', 50, 50, 0)).toBe(true);
    expect(evaluateAlert('price_below', 50, 51, 0)).toBe(false);
  });

  it('change percent', () => {
    expect(evaluateAlert('change_pct_above', 5, 0, 5)).toBe(true);
    expect(evaluateAlert('change_pct_above', 5, 0, 4.9)).toBe(false);
    expect(evaluateAlert('change_pct_below', -3, 0, -3)).toBe(true);
    expect(evaluateAlert('change_pct_below', -3, 0, -2)).toBe(false);
  });

  it('rejects bad inputs', () => {
    expect(evaluateAlert(undefined, 1, 1, 1)).toBe(false);
    expect(evaluateAlert('price_above', Number.NaN, 1, 0)).toBe(false);
  });
});

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
