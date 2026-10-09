import { describe, expect, it } from 'vitest';
import { heatColor } from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.helpers';
import {
  bookHeatColumns,
  perpBookCells,
  chartBookWindow,
  chartBotMarks,
  chartProfileRead,
  chartSweepMarks,
  chartSweepVenue,
  chartTapeSpan,
  sweepSpanOutside,
  crowdTopLines,
  crowdTopSeries,
  candleFootprintSpan,
  candleFootprintView,
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
  liquidationBucketBars,
  liquidationProfile,
  openInterestLines,
  anchoredChange,
  anchoredVenueSeries,
  chartAnchors,
  chartPaneBucket,
  chartPaneFromSec,
  crowdGapLines,
  crowdStoredLines,
  venuesWithoutTop,
} from './chartDesk';

describe('chart desk mappers', () => {
  it('picks a book window from the visible span', () => {
    expect(chartBookWindow(4 * 60)).toBe(5 * 60);
    expect(chartBookWindow(20 * 60)).toBe(30 * 60);
    expect(chartBookWindow(2 * 3600)).toBe(4 * 3600);
    expect(chartBookWindow(10 * 86400)).toBe(30 * 86400);
  });

  it('rounds the tape span to 15 minutes and caps it at 30 days', () => {
    expect(chartTapeSpan(60, 1_000, 1_000 + 50 * 60)).toEqual({ from: 900, to: 4_500 });
    const from = 1_700_000_000;
    const to = from + 40 * 86400;
    const span = chartTapeSpan(3600, from, to);
    expect(span).not.toBeNull();
    expect((span?.to ?? 0) - (span?.from ?? 0)).toBeLessThanOrEqual(30 * 86400);
  });

  it('draws combined profile lines only when every venue reaches the open', () => {
    const partial = chartProfileRead(
      {
        definite: false,
        missing: ['okx'],
        venues: [{ exchange: 'binance', poc: { price: '100', high: '102' }, valueArea: { low: '90', high: '110' } }],
      },
      'binance',
    );
    expect(partial.exchange).toBe('binance');
    expect(partial.definite).toBe(false);
    expect(partial.lines.map((line) => line.role)).toEqual(['poc', 'vaHigh', 'vaLow']);
    expect(partial.lines[0]?.price).toBe(101);

    const combined = chartProfileRead(
      {
        definite: true,
        missing: [],
        combined: { poc: { price: '200', high: '200' }, valueArea: { low: '180', high: '220' } },
        venues: [{ exchange: 'binance', poc: { price: '1', high: '1' }, valueArea: { low: '1', high: '2' } }],
      },
      'binance',
    );
    expect(combined.exchange).toBe('combined');
    expect(combined.definite).toBe(true);
    expect(combined.lines.find((line) => line.role === 'poc')?.price).toBe(200);
  });

  it('keeps sweep and bot marks that overlap the candles on screen', () => {
    expect(chartSweepVenue('bybit')).toBe('bybit');
    expect(chartSweepVenue('coinbase')).toBe('binance');
    const span = { from: 1_000, to: 5_000 };
    expect(
      chartSweepMarks(
        {
          venues: [
            {
              exchange: 'binance',
              sweeps: [
                { side: 'high', piercedAt: '1970-01-01T00:30:00.000Z' },
                { side: 'low', piercedAt: '1970-01-01T02:00:00.000Z' },
              ],
            },
          ],
        },
        'binance',
        span,
      ).map((mark) => mark.kind),
    ).toEqual(['sweep-high']);
    expect(
      chartBotMarks(
        {
          clusters: [
            { side: 'buy', count: 4, from: '1970-01-01T00:40:00.000Z', to: '1970-01-01T00:41:00.000Z' },
            { side: 'sell', count: 6, from: '1970-01-01T03:00:00.000Z', to: '1970-01-01T03:01:00.000Z' },
          ],
        },
        span,
      ),
    ).toEqual([{ time: 2_400, kind: 'bot-buy', count: 4 }]);
    expect(sweepSpanOutside({ from: 0, to: 10 }, 8 * 86400)).toBe(true);
    expect(sweepSpanOutside(span, 5_000)).toBe(false);
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

  it('paints perp bid and ask rings around the stored mid', () => {
    const cells = perpBookCells({
      stepSec: 60,
      columns: [
        {
          time: '2026-10-09T12:00:00Z',
          mid: '100',
          bands: [
            { rangePct: '0.1', bid: '10', ask: '5' },
            { rangePct: '1', bid: '40', ask: '20' },
          ],
        },
      ],
    });
    const bids = cells.filter((cell) => cell.priceHi <= 100);
    const asks = cells.filter((cell) => cell.priceLo >= 100);
    expect(bids.length).toBe(2);
    expect(asks.length).toBe(2);
    expect(bids.some((cell) => cell.priceLo < 99.5)).toBe(true);
    expect(cells.every((cell) => cell.stepSec === 60)).toBe(true);
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

  it('snaps the candle footprint to 8-candle blocks and keeps the newest 400', () => {
    const block = candleFootprintSpan(
      3600,
      Date.parse('2026-10-05T10:10:00Z') / 1000,
      Date.parse('2026-10-05T13:40:00Z') / 1000,
    );
    expect(block).toEqual({
      from: Date.parse('2026-10-05T08:00:00Z') / 1000,
      to: Date.parse('2026-10-05T16:00:00Z') / 1000,
    });
    expect(
      candleFootprintSpan(
        3600,
        Date.parse('2026-10-05T11:05:00Z') / 1000,
        Date.parse('2026-10-05T14:10:00Z') / 1000,
      ),
    ).toEqual(block);
    expect(candleFootprintSpan(3600, 0, 500 * 3600)).toEqual({
      from: 104 * 3600,
      to: 504 * 3600,
    });
    expect(candleFootprintSpan(0, 1, 2)).toBeNull();
    expect(candleFootprintSpan(Number.NaN, 1, 2)).toBeNull();
    expect(candleFootprintSpan(60, 10, 10)).toBeNull();
  });

  it('draws combined candle rows only when the book is definite', () => {
    const drawn = candleFootprintView(
      {
        definite: true,
        stepSec: 3600,
        truncated: true,
        missing: [],
        combined: {
          candles: [
            {
              time: '2026-10-05T10:00:00Z',
              rows: [
                {
                  price: '100',
                  high: '100.05',
                  buyVolume: '8',
                  sellVolume: '2',
                  isPoc: true,
                  imbalance: 'buy',
                },
                { price: '101', high: '101.05', buyVolume: '0', sellVolume: '0' },
              ],
            },
          ],
        },
      },
      'binance',
    );
    expect(drawn.definite).toBe(true);
    expect(drawn.truncated).toBe(true);
    expect(drawn.exchange).toBe('');
    expect(drawn.candles).toEqual([
      {
        time: Date.parse('2026-10-05T10:00:00Z') / 1000,
        rows: [{ price: 100, high: 100.05, buy: 8, sell: 2, poc: true, imbalance: 'buy' }],
      },
    ]);

    const withheld = candleFootprintView(
      {
        definite: false,
        missing: ['okx'],
        stepSec: 3600,
        combined: {
          candles: [
            {
              time: '2026-10-05T10:00:00Z',
              rows: [{ price: '1', high: '2', buyVolume: '9', sellVolume: '1' }],
            },
          ],
        },
        venues: [
          {
            exchange: 'binance',
            candles: [
              {
                time: '2026-10-05T10:00:00Z',
                rows: [{ price: '100', high: '101', buyVolume: '5', sellVolume: '4' }],
              },
            ],
          },
        ],
      },
      'binance',
    );
    expect(withheld.definite).toBe(false);
    expect(withheld.missing).toEqual(['okx']);
    expect(withheld.exchange).toBe('binance');
    expect(withheld.candles[0]?.rows[0]?.buy).toBe(5);

    const other = candleFootprintView(
      {
        definite: false,
        missing: ['binance'],
        venues: [
          { exchange: 'binance', candles: [] },
          {
            exchange: 'bybit',
            candles: [
              {
                time: '2026-10-05T10:00:00Z',
                rows: [{ price: '3', high: '4', buyVolume: '1', sellVolume: '1' }],
              },
            ],
          },
        ],
      },
      'binance',
    );
    expect(other.exchange).toBe('bybit');
    expect(other.candles).toHaveLength(1);
    expect(candleFootprintView(undefined, 'binance').candles).toEqual([]);
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

  it('picks the stored step from the candle span', () => {
    expect(chartPaneBucket(24 * 3600)).toBe('5m');
    expect(chartPaneBucket(24 * 3600 + 1)).toBe('30m');
    expect(chartPaneBucket(7 * 86400)).toBe('30m');
    expect(chartPaneBucket(7 * 86400 + 1)).toBe('2h');
    expect(chartPaneBucket(30 * 86400)).toBe('2h');
    expect(chartPaneBucket(30 * 86400 + 1)).toBe('4h');
  });

  it('floors the pane start onto the step and keeps it inside 90 days', () => {
    const now = 1_000_000_000;
    expect(chartPaneFromSec(now - 3 * 3600, now, '5m') % 300).toBe(0);
    const capped = chartPaneFromSec(now - 200 * 86400, now, '4h');
    expect(capped).toBeGreaterThanOrEqual(now - 90 * 86400);
    expect(now - capped).toBeLessThanOrEqual(90 * 86400);
    expect(capped % (4 * 3600)).toBe(0);
  });

  it('anchors a level at the last sample that reached the open', () => {
    const anchor = Date.parse('2026-10-08T00:00:00Z') / 1000;
    expect(
      anchoredChange(
        [
          { time: anchor - 3600, value: 100 },
          { time: anchor + 3600, value: 140 },
        ],
        anchor,
      ),
    ).toEqual([
      { time: anchor, value: 0 },
      { time: anchor + 3600, value: 40 },
    ]);
    expect(anchoredChange([{ time: anchor + 60, value: 10 }], anchor)).toEqual([]);
  });

  it('plots stored crowd and top, and names Bybit when it has no top share', () => {
    const rows = crowdStoredLines([
      {
        exchange: 'binance',
        sampledAt: '2026-10-08T12:00:00Z',
        longPct: '48',
        sizeLongPct: '61',
        topAccLongPct: '10',
      },
      { exchange: 'bybit', sampledAt: '2026-10-08T12:00:00Z', longPct: '55' },
      { exchange: 'okx', sampledAt: '2026-10-08T12:00:00Z', longPct: '52', topAccLongPct: '70' },
    ]);
    expect(rows.map((row) => row.exchange)).toEqual(['binance', 'bybit', 'okx']);
    expect(rows[0]?.top[0]?.value).toBe(61);
    expect(rows[1]?.top).toEqual([]);
    expect(rows[2]?.top[0]?.value).toBe(70);
    expect(venuesWithoutTop(rows)).toEqual(['bybit']);
    expect(crowdGapLines(rows).map((line) => line.exchange)).toEqual(['binance', 'okx']);
    expect(crowdGapLines(rows)[0]?.data[0]?.value).toBe(13);
  });

  it('draws the day and 4-hour change and names a venue that has not reached the open', () => {
    const day = Date.parse('2026-10-08T00:00:00Z') / 1000;
    const block = Date.parse('2026-10-08T12:00:00Z') / 1000;
    const read = anchoredVenueSeries(
      [
        {
          exchange: 'binance',
          data: [
            { time: day - 3600, value: 100 },
            { time: block, value: 130 },
          ],
        },
        {
          exchange: 'okx',
          data: [{ time: block + 3600, value: 80 }],
        },
      ],
      { day, block },
      { day: (exchange) => `${exchange} day`, block: (exchange) => `${exchange} 4h` },
      3,
      'oi',
    );
    expect(read.series.map((line) => line.id)).toEqual(['oi-day-binance', 'oi-4h-binance']);
    expect(read.series[0]?.data[0]?.value).toBe(0);
    expect(read.missingDay).toEqual(['okx']);
    expect(read.missingBlock).toEqual(['okx']);
    expect(chartAnchors(block + 60).block).toBe(block);
  });

  it('sums liquidation steps across venues', () => {
    const bars = liquidationBucketBars([
      { time: '2026-10-08T10:00:00Z', side: 'short', notional: '5', exchange: 'binance' },
      { time: '2026-10-08T10:00:00Z', side: 'long', notional: '2', exchange: 'bybit' },
      { time: '2026-10-08T10:30:00Z', side: 'long', notional: '4', exchange: 'okx' },
    ]);
    expect(bars).toEqual([
      { time: Date.parse('2026-10-08T10:00:00Z') / 1000, value: 3 },
      { time: Date.parse('2026-10-08T10:30:00Z') / 1000, value: -4 },
    ]);
  });
});
