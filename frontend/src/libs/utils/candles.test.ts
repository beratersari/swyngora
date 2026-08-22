import { describe, expect, it } from 'vitest';
import {
  apiCandlesToChart,
  carryForwardCandlesUntil,
  filterValidApiCandles,
  mergeCandleHistory,
  oldestCandleOpenTimeMs,
  preferLongerCandleSeries,
  trimCandlesToMax,
  type ApiCandle,
} from './candles';

function c(openTime: string, close = '1'): ApiCandle {
  return {
    openTime,
    open: close,
    high: close,
    low: close,
    close,
    volume: '0',
  };
}

describe('apiCandlesToChart', () => {
  it('maps valid candles to chart points with second timestamps', () => {
    const out = apiCandlesToChart([
      {
        openTime: '2024-01-01T00:00:00.000Z',
        open: '1',
        high: '2',
        low: '0.5',
        close: '1.5',
        volume: '10',
      },
    ]);
    expect(out).toHaveLength(1);
    expect(out[0]?.time).toBe(Math.floor(Date.parse('2024-01-01T00:00:00.000Z') / 1000));
    expect(out[0]?.close).toBe(1.5);
  });

  it('drops invalid openTime or non-numeric OHLC', () => {
    expect(
      apiCandlesToChart([
        { openTime: 'bad', open: '1', high: '1', low: '1', close: '1', volume: '0' },
        {
          openTime: '2024-01-01T00:00:00Z',
          open: 'x',
          high: '1',
          low: '1',
          close: '1',
          volume: '0',
        },
      ]),
    ).toEqual([]);
  });

  it('handles empty and mixed valid/invalid', () => {
    expect(apiCandlesToChart([])).toEqual([]);
    const mixed = apiCandlesToChart([
      { openTime: 'bad', open: '1', high: '1', low: '1', close: '1', volume: '0' },
      {
        openTime: '2024-01-01T00:00:00Z',
        open: '1',
        high: '2',
        low: '0.5',
        close: '1.5',
        volume: '1',
      },
    ]);
    expect(mixed).toHaveLength(1);
  });
});

describe('mergeCandleHistory', () => {
  it('merges by openTime, newer wins, sorted ascending', () => {
    const older = [c('2024-01-01T00:00:00Z', '1'), c('2024-01-01T01:00:00Z', '2')];
    const newer = [c('2024-01-01T01:00:00Z', '9'), c('2024-01-01T02:00:00Z', '3')];
    const out = mergeCandleHistory(older, newer);
    expect(out.map((x) => x.close)).toEqual(['1', '9', '3']);
  });
});

describe('oldestCandleOpenTimeMs', () => {
  it('returns earliest openTime ms', () => {
    const ms = oldestCandleOpenTimeMs([
      c('2024-01-02T00:00:00Z'),
      c('2024-01-01T00:00:00Z'),
    ]);
    expect(ms).toBe(Date.parse('2024-01-01T00:00:00Z'));
  });

  it('returns null for empty', () => {
    expect(oldestCandleOpenTimeMs([])).toBeNull();
  });
});

describe('trimCandlesToMax', () => {
  it('keeps newest bars when over max', () => {
    const bars = [
      c('2024-01-01T00:00:00Z'),
      c('2024-01-01T01:00:00Z'),
      c('2024-01-01T02:00:00Z'),
    ];
    expect(trimCandlesToMax(bars, 2).map((x) => x.openTime)).toEqual([
      '2024-01-01T01:00:00Z',
      '2024-01-01T02:00:00Z',
    ]);
  });
});

describe('preferLongerCandleSeries', () => {
  it('uses the longer live window and ignores empty sides', () => {
    const short = [c('2024-01-01T00:00:00Z')];
    const long = [c('2024-01-01T00:00:00Z'), c('2024-01-01T01:00:00Z')];
    expect(preferLongerCandleSeries(short, long)).toBe(long);
    expect(preferLongerCandleSeries(short, undefined)).toBe(short);
    expect(preferLongerCandleSeries(undefined, undefined)).toBeUndefined();
  });
});

describe('carryForwardCandlesUntil', () => {
  it('repeats the last close through the requested end', () => {
    const bars = [c('2026-08-17T02:00:00Z', '0.0077')];
    const until = Date.parse('2026-08-17T05:00:00Z');
    const got = carryForwardCandlesUntil(bars, until, 3600);
    expect(got.map((x) => x.openTime)).toEqual([
      '2026-08-17T02:00:00Z',
      '2026-08-17T03:00:00.000Z',
      '2026-08-17T04:00:00.000Z',
    ]);
    expect(got[2]?.close).toBe('0.0077');
    expect(got[2]?.volume).toBe('0');
    expect(Number(got[2]?.high)).toBeGreaterThan(Number(got[2]?.low));
  });

  it('does nothing without a positive interval', () => {
    const bars = [c('2026-08-17T02:00:00Z')];
    expect(carryForwardCandlesUntil(bars, Date.now(), 0)).toHaveLength(1);
  });

  it('keeps leftover session bars after official midnight halt and continues past that day', () => {
    const halt = Date.parse('2026-08-17T00:00:00Z');
    const live: ApiCandle[] = [
      { ...c('2026-08-16T23:00:00Z', '0.01'), high: '0.012', low: '0.009' },
      { ...c('2026-08-17T02:00:00Z', '0.0077'), high: '0.0081', low: '0.0064' },
    ];
    const until = Date.parse('2026-08-19T00:00:00Z');
    const got = carryForwardCandlesUntil(live, until, 3600);
    const afterHalt = got.filter((b) => Date.parse(b.openTime) >= halt);
    expect(afterHalt.length).toBeGreaterThan(20);
    expect(got.some((b) => b.openTime.startsWith('2026-08-18'))).toBe(true);
    const last = got[got.length - 1]!;
    expect(Number(last.high)).toBeGreaterThan(Number(last.low));
    expect(last.close).toBe('0.0077');
  });
});

describe('filterValidApiCandles', () => {
  it('drops incomplete rows', () => {
    expect(
      filterValidApiCandles([
        { openTime: '2024-01-01T00:00:00Z', open: '1', high: '1', low: '1', close: '1' },
        { openTime: 'bad', open: '1', high: '1', low: '1', close: '1' },
        undefined,
      ]),
    ).toHaveLength(1);
  });
});
