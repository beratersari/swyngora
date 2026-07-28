import { describe, expect, it } from 'vitest';
import {
  formatIndicator,
  indicatorPointsToEmaLine,
  indicatorPointsToRsiLine,
  rsiBandKey,
  rsiBandLabel,
  rsiTone,
  sortedEmaKeys,
} from './indicators';

describe('formatIndicator', () => {
  it('formats finite numbers with default and custom digits', () => {
    expect(formatIndicator(55.1234)).toBe('55.12');
    expect(formatIndicator(55.1234, 4)).toMatch(/55\.1234/);
  });

  it('returns dash for nullish / non-finite', () => {
    expect(formatIndicator(null)).toBe('—');
    expect(formatIndicator(undefined)).toBe('—');
    expect(formatIndicator(Number.NaN)).toBe('—');
  });
});

describe('rsiTone / rsiBandKey alignment', () => {
  it('classifies oversold / overbought / neutral consistently', () => {
    expect(rsiTone(25)).toBe('success');
    expect(rsiBandKey(25)).toBe('oversold');
    expect(rsiBandLabel(25)).toBe('oversold');

    expect(rsiTone(80)).toBe('error');
    expect(rsiBandKey(80)).toBe('overbought');

    expect(rsiTone(50)).toBe('secondary');
    expect(rsiBandKey(50)).toBe('neutral');

    // Near-band values stay neutral (color matches band label)
    expect(rsiTone(35)).toBe('secondary');
    expect(rsiBandKey(35)).toBe('neutral');
    expect(rsiTone(65)).toBe('secondary');
    expect(rsiBandKey(65)).toBe('neutral');

    // Boundaries
    expect(rsiTone(30)).toBe('secondary');
    expect(rsiBandKey(30)).toBe('neutral');
    expect(rsiTone(70)).toBe('secondary');
    expect(rsiBandKey(70)).toBe('neutral');
    expect(rsiTone(29.9)).toBe('success');
    expect(rsiTone(70.1)).toBe('error');

    expect(rsiTone(null)).toBe('secondary');
    expect(rsiBandKey(null)).toBe('na');
    expect(rsiBandLabel(null)).toBe('n/a');
    expect(rsiTone(Number.NaN)).toBe('secondary');
  });
});

describe('indicatorPointsToRsiLine', () => {
  it('maps valid points and skips warm-up nulls / bad times', () => {
    const line = indicatorPointsToRsiLine([
      { openTime: '2024-01-01T00:00:00Z', rsi: null },
      { openTime: 'not-a-date', rsi: 50 },
      { openTime: '2024-01-01T01:00:00Z', rsi: 42.5 },
    ]);
    expect(line).toHaveLength(1);
    expect(line[0]?.value).toBe(42.5);
  });

  it('returns empty for missing points', () => {
    expect(indicatorPointsToRsiLine(undefined)).toEqual([]);
    expect(indicatorPointsToRsiLine([])).toEqual([]);
  });
});

describe('indicatorPointsToEmaLine / sortedEmaKeys', () => {
  it('extracts EMA series and sorts keys', () => {
    const points: {
      openTime: string;
      ema: Record<string, number>;
    }[] = [
      { openTime: '2024-01-01T00:00:00Z', ema: { '12': 100, '26': 99 } },
      { openTime: '2024-01-01T01:00:00Z', ema: { '12': 101, '26': 99.5 } },
      { openTime: 'bad', ema: { '12': 102 } },
      { openTime: '2024-01-01T02:00:00Z', ema: { '12': Number.NaN } },
    ];
    expect(indicatorPointsToEmaLine(points, '12')).toHaveLength(2);
    expect(indicatorPointsToEmaLine(points, '99')).toEqual([]);
    expect(indicatorPointsToEmaLine(undefined, '12')).toEqual([]);
    expect(sortedEmaKeys({ '26': 1, '12': 2 })).toEqual(['12', '26']);
    expect(sortedEmaKeys(undefined)).toEqual([]);
    expect(sortedEmaKeys({})).toEqual([]);
  });
});
