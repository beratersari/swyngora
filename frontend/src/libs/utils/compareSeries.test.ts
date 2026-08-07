import { describe, expect, it } from 'vitest';
import {
  closesToPercentSeries,
  parseComparePairsParam,
  serializeComparePairs,
  MAX_COMPARE_PAIRS,
} from './compareSeries';

describe('closesToPercentSeries', () => {
  it('normalizes to percent from first close', () => {
    const out = closesToPercentSeries([
      { time: 1, close: 100 },
      { time: 2, close: 110 },
      { time: 3, close: 90 },
    ]);
    expect(out).toHaveLength(3);
    expect(out[0]!.value).toBeCloseTo(0);
    expect(out[1]!.value).toBeCloseTo(10);
    expect(out[2]!.value).toBeCloseTo(-10);
  });

  it('returns empty when no finite closes', () => {
    expect(closesToPercentSeries([])).toEqual([]);
    expect(closesToPercentSeries([{ time: 1, close: Number.NaN }])).toEqual([]);
  });
});

describe('compare pairs URL helpers', () => {
  it('parses and caps pairs', () => {
    const raw = 'binance:BTCUSDT,coinbase:ETH-USD,bybit:SOLUSDT,binance:X';
    const pairs = parseComparePairsParam(raw);
    expect(pairs).toHaveLength(MAX_COMPARE_PAIRS);
    expect(pairs[0]).toEqual({ exchange: 'binance', symbol: 'BTCUSDT' });
    expect(serializeComparePairs(pairs)).toContain('binance:BTCUSDT');
  });

  it('dedupes and ignores junk', () => {
    expect(parseComparePairsParam('binance:BTCUSDT,BINANCE:btcusdt,nocolon')).toHaveLength(1);
    expect(parseComparePairsParam('')).toEqual([]);
  });
});
