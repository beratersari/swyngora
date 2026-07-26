import { describe, expect, it } from 'vitest';
import {
  formatIndicator,
  indicatorPointsToEmaLine,
  indicatorPointsToRsiLine,
  rsiBandLabel,
  rsiTone,
  sortedEmaKeys,
} from './indicators';

describe('formatIndicator', () => {
  it('formats finite numbers', () => {
    expect(formatIndicator(55.1234)).toBe('55.12');
  });

  it('returns dash for nullish / non-finite', () => {
    expect(formatIndicator(null)).toBe('—');
    expect(formatIndicator(undefined)).toBe('—');
    expect(formatIndicator(Number.NaN)).toBe('—');
  });
});

describe('rsiTone / rsiBandLabel', () => {
  it('classifies bands', () => {
    expect(rsiTone(25)).toBe('success');
    expect(rsiBandLabel(25)).toBe('oversold');
    expect(rsiTone(80)).toBe('error');
    expect(rsiBandLabel(80)).toBe('overbought');
    expect(rsiTone(50)).toBe('secondary');
    expect(rsiBandLabel(50)).toBe('neutral');
  });
});

describe('indicatorPointsToRsiLine', () => {
  it('maps valid points and skips warm-up nulls', () => {
    const line = indicatorPointsToRsiLine([
      { openTime: '2024-01-01T00:00:00Z', rsi: null },
      { openTime: '2024-01-01T01:00:00Z', rsi: 42.5 },
    ]);
    expect(line).toHaveLength(1);
    expect(line[0]?.value).toBe(42.5);
    expect(line[0]?.time).toBe(Math.floor(Date.parse('2024-01-01T01:00:00Z') / 1000));
  });
});

describe('indicatorPointsToEmaLine / sortedEmaKeys', () => {
  it('extracts EMA series and sorts keys', () => {
    const points = [
      { openTime: '2024-01-01T00:00:00Z', ema: { '12': 100, '26': 99 } },
      { openTime: '2024-01-01T01:00:00Z', ema: { '12': 101, '26': 99.5 } },
    ];
    expect(indicatorPointsToEmaLine(points, '12')).toHaveLength(2);
    expect(sortedEmaKeys({ '26': 1, '12': 2 })).toEqual(['12', '26']);
  });
});
