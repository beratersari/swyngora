import { describe, expect, it } from 'vitest';
import type { MarketCvd, MarketLiquidationLevels } from '@/libs/api';
import {
  anchoredCvdByCandle,
  cvdAnchorStartsLate,
  cvdBarSeconds,
  cvdRangeForAnchor,
  deltaByCandle,
  formatUtcMinute,
  snapCandleOpen,
  futuresCvdSeries,
  futuresSizeSeries,
  sizedCvdByCandle,
  spotCvdSeries,
  liquidationChartLevels,
  parseTapeNumber,
  runningCvdByCandle,
  utc4hOpenSec,
  utcDayOpenSec,
} from './chartTape';

describe('chartTape', () => {
  it('parses signed quantities', () => {
    expect(parseTapeNumber('+12.5')).toBe(12.5);
    expect(parseTapeNumber('-3')).toBe(-3);
    expect(parseTapeNumber('nope')).toBeNull();
  });

  it('opens the UTC day and 4h block', () => {
    const at = Math.floor(Date.parse('2026-09-28T10:30:00Z') / 1000);
    expect(utcDayOpenSec(at)).toBe(Math.floor(Date.parse('2026-09-28T00:00:00Z') / 1000));
    expect(utc4hOpenSec(at)).toBe(Math.floor(Date.parse('2026-09-28T08:00:00Z') / 1000));
  });

  it('sums 5-minute deltas into the candle that contains them', () => {
    const hour = Math.floor(Date.parse('2026-09-28T10:00:00Z') / 1000);
    const bars = deltaByCandle(
      [{ time: hour }, { time: hour + 3600 }],
      [
        { time: hour, delta: 10 },
        { time: hour + 300, delta: -4 },
        { time: hour + 3600, delta: 7 },
      ],
      3600,
    );
    expect(bars).toEqual([
      { time: hour, value: 6 },
      { time: hour + 3600, value: 7 },
    ]);
  });

  it('carries the running CVD sum onto later candles', () => {
    const hour = Math.floor(Date.parse('2026-09-28T10:00:00Z') / 1000);
    const line = runningCvdByCandle(
      [{ time: hour }, { time: hour + 3600 }, { time: hour + 7200 }],
      [
        { time: hour, delta: 10, cvd: 10 },
        { time: hour + 1800, delta: -4, cvd: 6 },
        { time: hour + 7200, delta: 3, cvd: 9 },
      ],
      3600,
    );
    expect(line).toEqual([
      { time: hour, value: 6 },
      { time: hour + 3600, value: 6 },
      { time: hour + 7200, value: 9 },
    ]);
  });

  it('resets anchored CVD at the clock and carries the sum forward', () => {
    const day = Math.floor(Date.parse('2026-09-28T00:00:00Z') / 1000);
    const line = anchoredCvdByCandle(
      [
        { time: day - 3600 },
        { time: day },
        { time: day + 3600 },
        { time: day + 7200 },
      ],
      [
        { time: day - 3600, delta: 100 },
        { time: day, delta: 5 },
        { time: day + 3600, delta: -2 },
      ],
      day,
      3600,
    );
    expect(line).toEqual([
      { time: day, value: 5 },
      { time: day + 3600, value: 3 },
      { time: day + 7200, value: 3 },
    ]);
  });

  it('reads combined futures points and anchor clocks', () => {
    const report: MarketCvd = {
      combined: {
        points: [
          { time: '2026-09-28T09:55:00Z', delta: '+2' },
          { time: '2026-09-28T10:00:00Z', delta: '-1' },
        ],
        complete: true,
      anchors: [
          { anchor: 'day', from: '2026-09-28T00:00:00Z' },
          { anchor: '4h', from: '2026-09-28T08:00:00Z' },
        ],
      },
      venues: [
        { exchange: 'binance', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+9' }] },
        { exchange: 'bybit', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+4' }] },
        { exchange: 'okx', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+1' }] },
      ],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.source).toBe('combined');
    expect(series.withheld).toEqual([]);
    expect(series.points.map((p) => p.delta)).toEqual([2, -1]);
    expect(series.dayFrom).toBe(Math.floor(Date.parse('2026-09-28T00:00:00Z') / 1000));
    expect(series.blockFrom).toBe(Math.floor(Date.parse('2026-09-28T08:00:00Z') / 1000));
  });

  it('maps whale and retail deltas and skips a bar with no size', () => {
    const report: MarketCvd = {
      venues: [
        {
          exchange: 'binance',
          complete: true,
          points: [
            { time: '2026-09-28T10:00:00Z', delta: '+10', whaleDelta: '+100000', retailDelta: '-500' },
            { time: '2026-09-28T10:05:00Z', delta: '+3' },
            { time: '2026-09-28T10:10:00Z', delta: '+1', whaleDelta: '+5' },
            { time: '2026-09-28T10:15:00Z', delta: '0', whaleDelta: '0', retailDelta: '+2' },
          ],
        },
        { exchange: 'bybit', points: [] },
        { exchange: 'okx', points: [] },
      ],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.points.map((point) => point.whaleDelta)).toEqual([100000, undefined, undefined, 0]);
    expect(series.points.map((point) => point.retailDelta)).toEqual([-500, undefined, undefined, 2]);
    const size = futuresSizeSeries(report, 'binance');
    expect(size.source).toBe('binance');
    expect(size.withheld).toEqual(['bybit', 'okx']);
    expect(size.whale.map((point) => point.delta)).toEqual([100000, 0]);
    expect(size.whale.map((point) => point.cvd)).toEqual([100000, 100000]);
    expect(size.retail.map((point) => point.delta)).toEqual([-500, 2]);
    expect(size.retail.map((point) => point.cvd)).toEqual([-500, -498]);
  });

  it('uses combined size only when a combined bar was sized', () => {
    const venues: MarketCvd['venues'] = [
      { exchange: 'binance', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+9', whaleDelta: '+9', retailDelta: '0' }] },
      { exchange: 'bybit', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+4', whaleDelta: '0', retailDelta: '+4' }] },
      { exchange: 'okx', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+1', whaleDelta: '0', retailDelta: '+1' }] },
    ];
    const sized = futuresSizeSeries(
      {
        combined: {
          complete: true,
          points: [
            { time: '2026-09-28T10:00:00Z', delta: '+2', whaleDelta: '+1', retailDelta: '+1' },
            { time: '2026-09-28T10:05:00Z', delta: '+4' },
          ],
        },
        venues,
      },
      'binance',
    );
    expect(sized.source).toBe('combined');
    expect(sized.withheld).toEqual([]);
    expect(sized.whale.map((point) => point.delta)).toEqual([1]);
    const withheld = futuresSizeSeries(
      {
        combined: { complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+2' }] },
        venues,
      },
      'binance',
    );
    expect(futuresCvdSeries({ combined: { complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+2' }] }, venues }, 'binance').source).toBe('combined');
    expect(withheld.source).toBe('binance');
    expect(withheld.whale.map((point) => point.delta)).toEqual([9]);
    expect(withheld.withheld).toEqual(['bybit', 'okx']);
  });

  it('holds a sized bar for one step, then breaks instead of drawing zero', () => {
    const candles = [0, 60, 120, 180, 240, 300, 360].map((time) => ({ time }));
    const points = [
      { time: 0, delta: 10, cvd: 10 },
      { time: 360, delta: 4, cvd: 14 },
    ];
    expect(sizedCvdByCandle(candles, points, null, 60, 300)).toEqual([
      { time: 0, value: 10 },
      { time: 60, value: 10 },
      { time: 120, value: 10 },
      { time: 180, value: 10 },
      { time: 240, value: 10 },
      { time: 300, value: Number.NaN, gap: true },
      { time: 360, value: 14 },
    ]);
    expect(sizedCvdByCandle(candles, points, 360, 60, 300)).toEqual([{ time: 360, value: 4 }]);
    expect(
      sizedCvdByCandle(
        candles,
        [
          { time: 0, delta: 10, cvd: 10 },
          { time: 300, delta: 4, cvd: 14 },
        ],
        null,
        60,
        300,
      ).some((point) => point.gap),
    ).toBe(false);
  });

  it('draws the longer Binance tape when combined does not cover it', () => {
    const report: MarketCvd = {
      combined: {
        complete: false,
        points: [{ time: '2026-09-28T10:00:00Z', delta: '+1' }],
      },
      venues: [
        {
          exchange: 'binance',
          complete: true,
          points: [
            { time: '2026-09-21T10:00:00Z', delta: '+4' },
            { time: '2026-09-28T10:00:00Z', delta: '+2' },
          ],
          divergence: {
            kind: 'price_up_cvd_down',
            since: '2026-09-21T10:00:00Z',
            duration: '7d',
            durationSeconds: 7 * 24 * 3600,
          },
        },
        { exchange: 'bybit', points: [{ time: '2026-09-28T10:00:00Z', delta: '+9' }] },
        { exchange: 'okx', points: [] },
      ],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.source).toBe('binance');
    expect(series.points.map((p) => p.delta)).toEqual([4, 2]);
    expect(series.withheld).toEqual(['bybit', 'okx']);
    expect(series.split).toEqual({
      time: Math.floor(Date.parse('2026-09-21T10:00:00Z') / 1000),
      kind: 'price_up_cvd_down',
      duration: '7d',
      durationSeconds: 7 * 24 * 3600,
    });
  });

  it('ignores a window fallback that has no since', () => {
    const report: MarketCvd = {
      combined: {
        complete: true,
        points: [{ time: '2026-09-28T12:00:00Z', delta: '+1' }],
        divergence: { kind: 'price_down_cvd_up', since: '2026-09-28T12:00:00Z', duration: '5m', durationSeconds: 300 },
      },
      venues: [
        {
          exchange: 'binance',
          points: [
            { time: '2026-09-21T10:00:00Z', delta: '+4' },
            { time: '2026-09-28T10:00:00Z', delta: '+2' },
          ],
          divergence: { kind: 'price_up_cvd_down' },
        },
      ],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.source).toBe('binance');
    expect(series.split).toBeNull();
  });

  it('marks the combined run when combined covers the page venue', () => {
    const report: MarketCvd = {
      combined: {
        complete: true,
        points: [{ time: '2026-09-28T10:00:00Z', delta: '-1' }],
        divergence: {
          kind: 'price_down_cvd_up',
          since: '2026-09-28T09:25:00Z',
          duration: '35m',
          durationSeconds: 2100,
        },
      },
      venues: [
        { exchange: 'binance', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+9' }] },
        { exchange: 'bybit', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+4' }] },
        { exchange: 'okx', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+1' }] },
      ],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.source).toBe('combined');
    expect(series.split?.kind).toBe('price_down_cvd_up');
    expect(series.split?.duration).toBe('35m');
  });

  it('drops a run once a later bar moves price and CVD together', () => {
    const report: MarketCvd = {
      venues: [
        {
          exchange: 'binance',
          points: [
            { time: '2026-09-28T10:00:00Z', delta: '-4', priceChangePct: '-0.20' },
            { time: '2026-09-28T10:05:00Z', delta: '+9', priceChangePct: '+0.20' },
          ],
          divergence: {
            kind: 'price_down_cvd_up',
            since: '2026-09-28T10:00:00Z',
            lastAt: '2026-09-28T10:00:00Z',
            duration: '5m',
            durationSeconds: 300,
          },
        },
      ],
    };
    expect(futuresCvdSeries(report, 'binance').split).toBeNull();
  });

  it('keeps a run when only flat bars follow it', () => {
    const report: MarketCvd = {
      venues: [
        {
          exchange: 'binance',
          points: [
            { time: '2026-09-28T10:00:00Z', delta: '-4', priceChangePct: '+0.20' },
            { time: '2026-09-28T10:05:00Z', delta: '0', priceChangePct: '+0.01' },
          ],
          divergence: {
            kind: 'price_up_cvd_down',
            since: '2026-09-28T10:00:00Z',
            lastAt: '2026-09-28T10:00:00Z',
            duration: '5m',
            durationSeconds: 300,
          },
        },
      ],
    };
    expect(futuresCvdSeries(report, 'binance').split?.kind).toBe('price_up_cvd_down');
  });

  it('does not draw futures combined while OKX is missing', () => {
    const report: MarketCvd = {
      combined: {
        complete: true,
        points: [{ time: '2026-09-28T10:00:00Z', delta: '+3' }],
      },
      venues: [
        { exchange: 'binance', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+9' }] },
        { exchange: 'bybit', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+4' }] },
      ],
    };
    const series = futuresCvdSeries(report, 'coinbase');
    expect(series.source).toBe('binance');
    expect(series.withheld).toEqual(['okx']);
    expect(series.points.map((p) => p.delta)).toEqual([9]);
  });

  it('draws combined spot CVD only when Binance and Bybit both cover the window', () => {
    const covered = { time: '2026-09-28T10:00:00Z', delta: '+2' };
    const report: MarketCvd = {
      spotCombined: { complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+6' }] },
      spotVenues: [
        { exchange: 'binance', complete: true, points: [covered] },
        { exchange: 'bybit', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+4' }] },
      ],
    };
    expect(spotCvdSeries(report, 'binance').source).toBe('combined');
    expect(spotCvdSeries(report, 'binance').withheld).toEqual([]);
  });

  it('names Binance spot CVD on a Coinbase page when Bybit has not covered the window', () => {
    const report: MarketCvd = {
      spotCombined: {
        complete: true,
        points: [{ time: '2026-09-28T10:00:00Z', delta: '+6' }],
      },
      spotVenues: [
        {
          exchange: 'binance',
          complete: true,
          points: [{ time: '2026-09-28T10:00:00Z', delta: '+8' }],
        },
        { exchange: 'bybit', complete: false, points: [] },
      ],
    };
    const series = spotCvdSeries(report, 'binance');
    expect(series.source).toBe('binance');
    expect(series.withheld).toEqual(['bybit']);
    expect(series.points.map((p) => p.delta)).toEqual([8]);
  });

  it('names Bybit when the Bybit spot line is the one that has not covered the window', () => {
    const at = { time: '2026-09-28T10:00:00Z', delta: '+1' };
    const report: MarketCvd = {
      spotVenues: [
        { exchange: 'binance', complete: true, points: [{ time: '2026-09-28T10:00:00Z', delta: '+8' }] },
        { exchange: 'bybit', complete: false, points: [at] },
      ],
    };
    const series = spotCvdSeries(report, 'bybit');
    expect(series.source).toBe('bybit');
    expect(series.withheld).toEqual(['bybit']);
  });

  it('picks the strongest long below last and short above last per leverage', () => {
    const report: MarketLiquidationLevels = {
      kind: 'levels',
      lastPrice: '100',
      levels: [
        {
          price: '90',
          byLeverage: [
            { leverage: 100, longNotional: '5', shortNotional: '0' },
            { leverage: 50, longNotional: '1', shortNotional: '0' },
          ],
        },
        {
          price: '95',
          byLeverage: [{ leverage: 100, longNotional: '20', shortNotional: '0' }],
        },
        {
          price: '110',
          byLeverage: [{ leverage: 100, longNotional: '99', shortNotional: '8' }],
        },
        {
          price: '120',
          byLeverage: [{ leverage: 100, longNotional: '0', shortNotional: '3' }],
        },
      ],
    };
    const lines = liquidationChartLevels(report);
    const long100 = lines.find((line) => line.side === 'long' && line.leverages.includes(100));
    const short100 = lines.find((line) => line.side === 'short' && line.leverages.includes(100));
    expect(long100?.price).toBe(95);
    expect(short100?.price).toBe(110);
  });

  it('widens the CVD range to the candle and snaps the click onto that candle', () => {
    const now = Math.floor(Date.parse('2026-10-09T12:00:00Z') / 1000);
    const twoDays = now - 2 * 24 * 60 * 60;
    expect(cvdRangeForAnchor('24h', now - 3600, now)).toBe('24h');
    expect(cvdRangeForAnchor('24h', twoDays, now)).toBe('7d');
    expect(cvdRangeForAnchor('30d', twoDays, now)).toBe('30d');
    expect(cvdRangeForAnchor('24h', now - 40 * 24 * 60 * 60, now)).toBe('90d');
    expect(cvdBarSeconds('2h')).toBe(7200);
    const hour = Math.floor(Date.parse('2026-10-09T11:00:00Z') / 1000);
    expect(snapCandleOpen([{ time: hour }, { time: hour + 3600 }], hour + 120, 3600)).toBe(hour);
    expect(snapCandleOpen([{ time: hour }], hour - 10, 3600)).toBeNull();
    expect(formatUtcMinute(hour)).toBe('2026-10-09 11:00');
    expect(cvdAnchorStartsLate([{ time: hour }, { time: hour + 300 }], hour, 300)).toBe(false);
    expect(cvdAnchorStartsLate([{ time: hour + 3600 }], hour, 300)).toBe(true);
    expect(cvdAnchorStartsLate([], hour, 300)).toBe(true);
  });
});
