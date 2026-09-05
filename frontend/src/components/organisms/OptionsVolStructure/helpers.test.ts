import { describe, expect, it } from 'vitest';
import { formatSkewPts, ivBarPct, maxAtmIv } from './helpers';

describe('options vol helpers', () => {
  it('scales IV bars to the series max', () => {
    expect(maxAtmIv([{ atmIv: '40' }, { atmIv: '80' }])).toBe(80);
    expect(ivBarPct('40', 80)).toBe(50);
    expect(ivBarPct('0', 80)).toBe(0);
  });

  it('formats signed skew points', () => {
    expect(formatSkewPts('2.4')).toBe('+2.4');
    expect(formatSkewPts('-1.1')).toBe('-1.1');
    expect(formatSkewPts(undefined)).toBe('—');
  });
});
