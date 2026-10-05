import { describe, expect, it } from 'vitest';
import type { MarketCvd, MarketLiquidationLevels } from '@/libs/api';
import {
  anchoredCvdByCandle,
  deltaByCandle,
  futuresCvdSeries,
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
      venues: [{ exchange: 'binance', points: [{ time: '2026-09-28T10:00:00Z', delta: '+9' }] }],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.source).toBe('combined');
    expect(series.withheld).toEqual([]);
    expect(series.points.map((p) => p.delta)).toEqual([2, -1]);
    expect(series.dayFrom).toBe(Math.floor(Date.parse('2026-09-28T00:00:00Z') / 1000));
    expect(series.blockFrom).toBe(Math.floor(Date.parse('2026-09-28T08:00:00Z') / 1000));
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
        },
        { exchange: 'bybit', points: [{ time: '2026-09-28T10:00:00Z', delta: '+9' }] },
        { exchange: 'okx', points: [] },
      ],
    };
    const series = futuresCvdSeries(report, 'binance');
    expect(series.source).toBe('binance');
    expect(series.points.map((p) => p.delta)).toEqual([4, 2]);
    expect(series.withheld).toEqual(['bybit', 'okx']);
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
});
