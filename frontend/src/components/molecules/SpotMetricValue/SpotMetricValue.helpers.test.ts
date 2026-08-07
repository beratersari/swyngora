import { describe, expect, it } from 'vitest';
import { asTagList, formatSpotMetricDisplay } from './SpotMetricValue.helpers';

describe('formatSpotMetricDisplay', () => {
  it('formats price and change percent', () => {
    expect(formatSpotMetricDisplay('price', '42000.5')).toMatch(/42/);
    expect(formatSpotMetricDisplay('changePercent', '1.25')).toContain('%');
  });

  it('formats compact USD and trade counts', () => {
    expect(formatSpotMetricDisplay('compactUsd', 1_500_000)).toMatch(/M|1\.5/);
    expect(formatSpotMetricDisplay('tradeCount', 1200, 'binance')).not.toBe('—');
  });

  it('handles empty number and unknown formats', () => {
    expect(formatSpotMetricDisplay('number', null)).toBe('—');
    expect(formatSpotMetricDisplay('number', 12)).toBe('12');
    expect(formatSpotMetricDisplay('tags', ['a'])).toBe('—');
  });
});

describe('asTagList', () => {
  it('returns string arrays only', () => {
    expect(asTagList(['Meme', 'defi'])).toEqual(['Meme', 'defi']);
    expect(asTagList(undefined)).toEqual([]);
    expect(asTagList('x')).toEqual([]);
  });
});
