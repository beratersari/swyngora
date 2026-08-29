import { describe, expect, it } from 'vitest';
import {
  formatIndicator,
  indicatorPointsToEmaLine,
  indicatorPointsToRsiLine,
  mergeIndicatorPoints,
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

describe('rsiTone / rsiBandKey from API zone', () => {
  it('maps server zones onto color and i18n keys', () => {
    expect(rsiTone('oversold')).toBe('success');
    expect(rsiBandKey('oversold')).toBe('oversold');
    expect(rsiBandLabel('oversold')).toBe('oversold');

    expect(rsiTone('overbought')).toBe('error');
    expect(rsiBandKey('overbought')).toBe('overbought');

    expect(rsiTone('neutral')).toBe('secondary');
    expect(rsiBandKey('neutral')).toBe('neutral');

    expect(rsiTone('')).toBe('secondary');
    expect(rsiBandKey('')).toBe('na');
    expect(rsiBandLabel(null)).toBe('n/a');
    expect(rsiTone(undefined)).toBe('secondary');
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

describe('mergeIndicatorPoints', () => {
  it('merges by openTime with newer winning', () => {
    const older = [{ openTime: '2024-01-01T00:00:00Z', ema: { '12': 1 } }];
    const newer = [
      { openTime: '2024-01-01T00:00:00Z', ema: { '12': 2 } },
      { openTime: '2024-01-01T01:00:00Z', ema: { '12': 3 } },
    ];
    const merged = mergeIndicatorPoints(older, newer);
    expect(merged).toHaveLength(2);
    expect(merged[0]?.ema?.['12']).toBe(2);
    expect(merged[1]?.ema?.['12']).toBe(3);
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
