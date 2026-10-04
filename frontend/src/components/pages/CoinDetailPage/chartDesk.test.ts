import { describe, expect, it } from 'vitest';
import { heatColor } from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.helpers';
import {
  bookHeatColumns,
  chartBookWindow,
  crowdTopLines,
  crowdTopSeries,
  footprintLevels,
  footprintWindow,
  formatRecordedSpan,
  liqHeatCells,
  liqHeatNotes,
  openLevelDelta,
  openLevelNotes,
  openLevelSegments,
  recordedParts,
  fundingRateLines,
  futuresRecorded,
  liquidationNotionalByCandle,
  liquidationProfile,
  openInterestLines,
} from './chartDesk';

describe('chart desk mappers', () => {
  it('picks a book window from the visible span', () => {
    expect(chartBookWindow(4 * 60)).toBe(5 * 60);
    expect(chartBookWindow(20 * 60)).toBe(30 * 60);
    expect(chartBookWindow(2 * 3600)).toBe(4 * 3600);
    expect(chartBookWindow(10 * 86400)).toBe(30 * 86400);
  });

  it('picks a flow window from the visible span', () => {
    expect(footprintWindow(4 * 60)).toBe('5m');
    expect(footprintWindow(50 * 60)).toBe('1h');
    expect(footprintWindow(5 * 3600)).toBe('1d');
    expect(footprintWindow(10 * 86400)).toBe('30d');
    expect(footprintWindow(5 * 86400)).toBe('7d');
  });

  it('keeps the largest book levels and thins columns', () => {
    const columns = Array.from({ length: 180 }, (_, i) => ({
      t: new Date(Date.UTC(2026, 8, 29, 12, 0, i)).toISOString(),
      bids: [
        { price: '100', notional: '1' },
        { price: '99', notional: '50' },
      ],
      asks: [{ price: '101', notional: '20' }],
    }));
    const cols = bookHeatColumns({ columns });
    expect(cols.length).toBeLessThanOrEqual(90);
    expect(cols[0]?.bids[0]?.price).toBe(99);
    expect(cols[0]?.asks[0]?.notional).toBe(20);
  });

  it('reads combined footprint rows', () => {
    const levels = footprintLevels({
      combined: {
        levels: [
          { price: '100', buyVolume: '8', sellVolume: '2', volume: '10' },
          { price: '101', buyVolume: '0', sellVolume: '0', volume: '0' },
        ],
      },
    });
    expect(levels).toEqual([{ price: 100, buy: 8, sell: 2 }]);
  });

  it('paints heatmap cells and skips an empty combined column', () => {
    const cells = liqHeatCells(
      {
        stepSec: 1800,
        priceStep: 10,
        prices: [110, 100],
        times: ['2026-09-29T12:00:00Z', '2026-09-29T12:30:00Z'],
        combined: {
          totals: [
            [0, 8],
            [5, 5],
          ],
          maxIntensity: 8,
          hasData: [true, false],
        },
      },
      'combined',
      'totals',
    );
    expect(cells).toHaveLength(1);
    expect(cells[0]).toMatchObject({
      time: Date.parse('2026-09-29T12:00:00Z') / 1000,
      stepSec: 1800,
      priceHi: 105,
      priceLo: 95,
    });
    expect(cells[0]?.color).toBe(heatColor(1, 'totals'));
    const notes = liqHeatNotes(
      {
        leverageMix: [{ leverage: 10, weight: 0.25 }],
        missingVenues: ['okx'],
        okx: { accountBlend: 'even' },
      },
      'combined',
    );
    expect(notes.mix).toBe('10× 25%');
    expect(notes.missingVenues).toEqual(['okx']);
    expect(notes.evenVenues).toEqual(['okx']);
  });

  it('paints a measured print and skips a quiet recorded column', () => {
    const cells = liqHeatCells(
      {
        kind: 'prints',
        stepSec: 1800,
        priceStep: 10,
        prices: [110, 100],
        times: ['2026-09-29T12:00:00Z', '2026-09-29T12:30:00Z'],
        combined: {
          totals: [
            [4, 0],
            [0, 0],
          ],
          maxIntensity: 4,
          hasData: [true, true],
        },
      },
      'combined',
      'totals',
    );
    expect(cells).toHaveLength(1);
    expect(cells[0]).toMatchObject({
      time: Date.parse('2026-09-29T12:00:00Z') / 1000,
      priceHi: 115,
      priceLo: 105,
    });
  });

  it('draws open bands until they are hit and keeps the uncapped delta', () => {
    const report = {
      leverageMix: [{ leverage: 10, weight: 0.25 }],
      missingVenues: [] as string[],
      binance: {
        exchange: 'binance',
        definite: true,
        segments: [
          {
            side: 'long' as const,
            price: 90_000,
            notional: 400_000,
            openedAt: '2026-09-29T12:00:00Z',
            closedAt: '0001-01-01T00:00:00Z',
            status: 'open' as const,
          },
          {
            side: 'short' as const,
            price: 110_000,
            notional: 100_000,
            openedAt: '2026-09-29T12:00:00Z',
            closedAt: '2026-09-29T13:00:00Z',
            status: 'hit' as const,
          },
          {
            side: 'long' as const,
            price: 80_000,
            notional: 50_000,
            openedAt: '2026-09-29T12:00:00Z',
            closedAt: '2026-09-29T14:00:00Z',
            status: 'closed' as const,
          },
        ],
        cumulative: [
          { time: '2026-09-29T12:00:00Z', delta: 100 },
          { time: '2026-09-29T13:00:00Z', delta: -40 },
        ],
      },
      combined: {
        exchange: 'combined',
        definite: false,
        segments: [],
        cumulative: [],
      },
    };
    expect(openLevelSegments(report, 'binance').map((level) => level.status)).toEqual(['open', 'hit']);
    expect(openLevelSegments(report, 'binance')[0]?.closedAt).toBeUndefined();
    expect(openLevelSegments(report, 'binance')[1]?.closedAt).toBe(
      Date.parse('2026-09-29T13:00:00Z') / 1000,
    );
    expect(openLevelDelta(report, 'binance').map((point) => point.value)).toEqual([100, -40]);
    expect(openLevelSegments(report, 'combined')).toEqual([]);
    expect(openLevelDelta(report, 'combined')).toEqual([]);
    expect(openLevelNotes(report, 'combined')).toMatchObject({
      mix: '10× 25%',
      withheld: true,
    });
    expect(openLevelNotes(undefined, 'combined').withheld).toBe(false);
  });

  it('keeps the largest liquidation pockets on each side', () => {
    const bars = liquidationProfile({
      kind: 'levels',
      levels: [
        { price: '90', longNotional: '5', shortNotional: '0' },
        { price: '110', longNotional: '0', shortNotional: '9' },
      ],
    });
    expect(bars).toEqual([
      { price: 90, notional: 5, side: 'long' },
      { price: 110, notional: 9, side: 'short' },
    ]);
  });

  it('plots crowd and top-trader long percent', () => {
    const lines = crowdTopLines({
      gapHistory: [{ time: '2026-09-29T12:00:00Z', crowdLongPct: '48', topLongPct: '61' }],
    });
    expect(lines.crowd).toHaveLength(1);
    expect(lines.top[0]?.value).toBe(61);
  });

  it('plots Binance and OKX crowd versus top as separate lines', () => {
    const series = crowdTopSeries(
      {
        venues: [
          {
            exchange: 'binance',
            gapHistory: [{ time: '2026-09-29T12:00:00Z', crowdLongPct: '48', topLongPct: '61' }],
          },
          {
            exchange: 'okx',
            gapHistory: [{ time: '2026-09-29T12:00:00Z', crowdLongPct: '52', topLongPct: '70' }],
          },
        ],
      },
      {
        crowd: 'Crowd',
        top: 'Top',
        venueCrowd: (venue) => `${venue} crowd`,
        venueTop: (venue) => `${venue} top`,
      },
      { crowd: '#111', top: '#222', crowdAlt: '#333', topAlt: '#444' },
      2,
    );
    expect(series.map((line) => line.id)).toEqual([
      'crowd-long-binance',
      'top-long-binance',
      'crowd-long-okx',
      'top-long-okx',
    ]);
    expect(series[2]?.data[0]?.value).toBe(52);
    expect(series[3]?.title).toBe('okx top');
  });

  it('plots open interest per venue and drops empty values', () => {
    const lines = openInterestLines([
      { exchange: 'binance', sampledAt: '2026-09-29T12:00:00Z', value: '100' },
      { exchange: 'bybit', sampledAt: '2026-09-29T12:05:00Z', value: '40' },
      { exchange: 'okx', sampledAt: '2026-09-29T12:05:00Z', value: '' },
    ]);
    expect(lines.map((line) => line.exchange)).toEqual(['binance', 'bybit']);
    expect(lines[0]?.data[0]?.value).toBe(100);
  });

  it('plots settled funding and ignores predicted prints when a settlement exists', () => {
    const lines = fundingRateLines([
      { exchange: 'binance', sampledAt: '2026-09-29T08:00:00Z', ratePct: '0.01', predicted: false },
      { exchange: 'binance', sampledAt: '2026-09-29T12:00:00Z', ratePct: '0.05', predicted: true },
      { exchange: 'okx', sampledAt: '2026-09-29T12:00:00Z', ratePct: '-0.02', predicted: true },
    ]);
    expect(lines[0]?.data).toEqual([{ time: Date.parse('2026-09-29T08:00:00Z') / 1000, value: 0.01 }]);
    expect(lines[1]?.exchange).toBe('okx');
    expect(lines[1]?.data[0]?.value).toBe(-0.02);
  });

  it('buckets liquidation notional onto candles, short positive and long negative', () => {
    const candles = [{ time: 1_000 }, { time: 1_060 }];
    const bars = liquidationNotionalByCandle(
      candles,
      [
        { time: '1970-01-01T00:16:40Z', side: 'short', notional: '5' },
        { time: '1970-01-01T00:16:50Z', side: 'long', notional: '2' },
        { time: '1970-01-01T00:17:45Z', side: 'long', notional: '4' },
      ],
      60,
    );
    expect(bars).toEqual([
      { time: 1_000, value: 3 },
      { time: 1_060, value: -4 },
    ]);
  });

  it('formats futures recording spans', () => {
    expect(formatRecordedSpan(90)).toBe('2m');
    expect(formatRecordedSpan(3 * 3600)).toBe('3h');
    expect(formatRecordedSpan(3 * 86400 + 2 * 3600)).toBe('3d 2h');
    expect(
      futuresRecorded({
        series: [
          { kind: 'futures_oi', exchange: 'binance', spanSec: 3600 },
          { kind: 'book', exchange: 'bybit', spanSec: 10 },
        ],
      }),
    ).toEqual([{ exchange: 'binance', spanSec: 3600 }]);
    expect(
      recordedParts({
        series: [
          { kind: 'flow', exchange: 'bybit', spanSec: 120 },
          { kind: 'futures_oi', exchange: 'binance', spanSec: 3600 },
          { kind: 'liquidations', exchange: 'okx', spanSec: 0 },
        ],
      }),
    ).toEqual([
      { kind: 'futures_oi', exchange: 'binance', spanSec: 3600 },
      { kind: 'flow', exchange: 'bybit', spanSec: 120 },
    ]);
  });
});
