import type {
  CandleChartFootprintLevel,
  CandleChartHeatColumn,
  CandleChartLiqBar,
  CandleChartLiqCell,
  CandleChartLowerSeries,
} from '@/components/molecules/CandleChartHost';
import {
  cellValue,
  evenBlendVenueIds,
  formatLeverageMix,
  heatColor,
  intensityFromNotional,
  pickGrid,
  pickMatrix,
} from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.helpers';
import type { LiqHeatSide, LiqHeatVenue } from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.types';
import type { ChartLinePoint } from '@/libs/utils';
import type {
  FlowAtPricesReport,
  HistorySpan,
  LiquidationHuntHeatmap,
  MarketLiquidationLevels,
  MarketLongShortRatio,
  OrderBookHeatmap,
} from '@/libs/api';
import { semanticColors } from '@/styles/tokens';
import { parseTapeNumber, unixSeconds } from './chartTape';

/** Pane order for Binance, Bybit, and OKX lines. */
export const FUTURES_PANE_VENUES = ['binance', 'bybit', 'okx'] as const;

/** Newest stored rows the chart panes request. Matches the OpenAPI limit. */
export const FUTURES_PANE_LIMIT = 1000;

/** Line color for one futures venue on a lower pane. */
export function futuresPaneColor(exchange: string): string {
  switch (exchange) {
    case 'bybit':
      return semanticColors.status.warning;
    case 'okx':
      return semanticColors.chart.rsi;
    default:
      return semanticColors.accent.default;
  }
}

/** One stored futures-history row. The chart uses only the fields each pane needs. */
export type FuturesPaneRow = {
  exchange?: string;
  sampledAt?: string;
  time?: string;
  value?: string;
  ratePct?: string;
  predicted?: boolean;
  side?: string;
  notional?: string;
};

const HEAT_COLUMNS = 90;
const HEAT_LEVELS = 24;
const FOOTPRINT_ROWS = 48;
const LIQ_ROWS = 12;

/** Map a visible candle span onto a flow-at-prices window. */
export function footprintWindow(spanSec: number): '5m' | '15m' | '1h' | '4h' | '1d' | '3d' | '7d' | '30d' {
  if (!Number.isFinite(spanSec) || spanSec <= 5 * 60) return '5m';
  if (spanSec <= 15 * 60) return '15m';
  if (spanSec <= 60 * 60) return '1h';
  if (spanSec <= 4 * 60 * 60) return '4h';
  if (spanSec <= 24 * 60 * 60) return '1d';
  if (spanSec <= 3 * 24 * 60 * 60) return '3d';
  if (spanSec <= 7 * 24 * 60 * 60) return '7d';
  return '30d';
}

function topBy<T>(rows: T[], n: number, score: (row: T) => number): T[] {
  return [...rows].sort((a, b) => score(b) - score(a)).slice(0, n);
}

/** Downsample a book heatmap so the candle pane can paint it. */
export function bookHeatColumns(heat: OrderBookHeatmap | undefined): CandleChartHeatColumn[] {
  const cols = heat?.columns ?? [];
  if (cols.length === 0) return [];
  const step = Math.max(1, Math.ceil(cols.length / HEAT_COLUMNS));
  const out: CandleChartHeatColumn[] = [];
  for (let i = 0; i < cols.length; i += step) {
    const col = cols[i];
    const time = unixSeconds(col?.t);
    if (time == null || !col) continue;
    const side = (levels: { price?: string; notional?: string }[] | undefined) =>
      topBy(levels ?? [], HEAT_LEVELS, (lv) => parseTapeNumber(lv.notional) ?? 0)
        .map((lv) => {
          const price = parseTapeNumber(lv.price);
          const notional = parseTapeNumber(lv.notional);
          if (price == null || notional == null || notional <= 0) return null;
          return { price, notional };
        })
        .filter((lv): lv is { price: number; notional: number } => lv != null);
    const bids = side(col.bids);
    const asks = side(col.asks);
    if (bids.length === 0 && asks.length === 0) continue;
    out.push({ time, bids, asks });
  }
  return out;
}

/** Buy versus sell rows for the footprint on the right of the chart. */
export function footprintLevels(report: FlowAtPricesReport | undefined): CandleChartFootprintLevel[] {
  const levels = report?.combined?.levels ?? report?.venues?.[0]?.levels ?? [];
  const ranked = topBy(levels, FOOTPRINT_ROWS, (lv) => parseTapeNumber(lv.volume) ?? 0);
  const out: CandleChartFootprintLevel[] = [];
  for (const lv of ranked) {
    const price = parseTapeNumber(lv.price);
    const buy = parseTapeNumber(lv.buyVolume) ?? 0;
    const sell = parseTapeNumber(lv.sellVolume) ?? 0;
    if (price == null || buy + sell <= 0) continue;
    out.push({ price, buy, sell });
  }
  return out.sort((a, b) => a.price - b.price);
}

export type { LiqHeatSide, LiqHeatVenue };

export type LiqHeatNotes = {
  mix: string;
  evenVenues: Array<'binance' | 'bybit' | 'okx'>;
  missingVenues: string[];
};

function columnStepSec(times: string[] | undefined, stepSec: number | undefined): number {
  if (stepSec != null && stepSec > 0) return stepSec;
  const a = unixSeconds(times?.[0]);
  const b = unixSeconds(times?.[1]);
  if (a != null && b != null && b > a) return b - a;
  return 60;
}

function priceHalfStep(prices: number[] | undefined, priceStep: number | undefined): number {
  if (priceStep != null && priceStep > 0) return priceStep / 2;
  const a = prices?.[0];
  const b = prices?.[1];
  if (a != null && b != null && a !== b) return Math.abs(a - b) / 2;
  return 0;
}

/**
 * Paint cells for the liquidation heatmap behind the candles.
 * A combined column with hasData false is left empty. Zero cells are skipped
 * so the chart background stays clear outside the hot zones.
 */
export function liqHeatCells(
  data: LiquidationHuntHeatmap | undefined,
  venue: LiqHeatVenue,
  side: LiqHeatSide,
): CandleChartLiqCell[] {
  const prices = data?.prices ?? [];
  const times = data?.times ?? [];
  if (prices.length === 0 || times.length === 0) return [];
  const grid = pickGrid(data, venue);
  const matrix = pickMatrix(grid, side);
  let peak = grid?.maxIntensity ?? 0;
  if (!(peak > 0)) {
    for (const col of matrix) {
      for (const value of col ?? []) peak = Math.max(peak, value);
    }
  }
  if (!(peak > 0)) return [];
  const half = priceHalfStep(prices, data?.priceStep);
  const stepSec = columnStepSec(times, data?.stepSec);
  const out: CandleChartLiqCell[] = [];
  for (let i = 0; i < times.length; i += 1) {
    if (grid?.hasData?.[i] === false) continue;
    const time = unixSeconds(times[i]);
    if (time == null) continue;
    for (let j = 0; j < prices.length; j += 1) {
      const notional = cellValue(matrix, i, j);
      const price = prices[j] ?? 0;
      if (notional <= 0 || !Number.isFinite(price) || price <= 0) continue;
      out.push({
        time,
        stepSec,
        priceHi: price + half,
        priceLo: Math.max(0, price - half),
        color: heatColor(intensityFromNotional(notional, peak), side),
      });
    }
  }
  return out;
}

/** Leverage mix, even-split venues, and venues missing from this window. */
export function liqHeatNotes(
  data: LiquidationHuntHeatmap | undefined,
  venue: LiqHeatVenue,
): LiqHeatNotes {
  return {
    mix: formatLeverageMix(data?.leverageMix),
    evenVenues: evenBlendVenueIds(data, venue),
    missingVenues: (data?.missingVenues ?? []).filter((id) => id.length > 0),
  };
}

/** Larger estimated pockets, kept next to the dashed liquidation lines. */
export function liquidationProfile(report: MarketLiquidationLevels | undefined): CandleChartLiqBar[] {
  const levels = report?.levels ?? [];
  if (levels.length === 0 || report?.kind === 'totals') return [];
  const longs = topBy(levels, LIQ_ROWS, (lv) => parseTapeNumber(lv.longNotional) ?? 0);
  const shorts = topBy(levels, LIQ_ROWS, (lv) => parseTapeNumber(lv.shortNotional) ?? 0);
  const out: CandleChartLiqBar[] = [];
  for (const lv of longs) {
    const price = parseTapeNumber(lv.price);
    const notional = parseTapeNumber(lv.longNotional) ?? 0;
    if (price == null || notional <= 0) continue;
    out.push({ price, notional, side: 'long' });
  }
  for (const lv of shorts) {
    const price = parseTapeNumber(lv.price);
    const notional = parseTapeNumber(lv.shortNotional) ?? 0;
    if (price == null || notional <= 0) continue;
    out.push({ price, notional, side: 'short' });
  }
  return out;
}

export type CrowdChartLabels = {
  crowd: string;
  top: string;
  venueCrowd?: (venue: string) => string;
  venueTop?: (venue: string) => string;
};

/** Crowd long percent and top-trader long percent, one point per stored print. */
export function crowdTopLines(report: MarketLongShortRatio | undefined): {
  crowd: ChartLinePoint[];
  top: ChartLinePoint[];
} {
  const crowd: ChartLinePoint[] = [];
  const top: ChartLinePoint[] = [];
  for (const pt of report?.gapHistory ?? []) {
    const time = unixSeconds(pt.time);
    const crowdPct = parseTapeNumber(pt.crowdLongPct);
    const topPct = parseTapeNumber(pt.topLongPct);
    if (time == null || crowdPct == null || topPct == null) continue;
    crowd.push({ time, value: crowdPct });
    top.push({ time, value: topPct });
  }
  crowd.sort((a, b) => a.time - b.time);
  top.sort((a, b) => a.time - b.time);
  return { crowd, top };
}

function crowdSources(report: MarketLongShortRatio | undefined): MarketLongShortRatio[] {
  const venues = (report?.venues ?? []).filter((venue) => (venue.gapHistory?.length ?? 0) > 0);
  if (venues.length > 0) {
    return venues.map((venue) => ({ gapHistory: venue.gapHistory, exchange: venue.exchange }));
  }
  if ((report?.gapHistory?.length ?? 0) > 0) return [report ?? {}];
  return [];
}

export function crowdTopSeries(
  report: MarketLongShortRatio | undefined,
  labels: CrowdChartLabels,
  colors: { crowd: string; top: string; crowdAlt?: string; topAlt?: string },
  pane: number,
): CandleChartLowerSeries[] {
  const sources = crowdSources(report);
  const out: CandleChartLowerSeries[] = [];
  sources.forEach((source, index) => {
    const lines = crowdTopLines(source);
    const venue = source.exchange ?? '';
    const suffix = sources.length > 1 && venue ? `-${venue}` : '';
    const crowdTitle = venue && labels.venueCrowd ? labels.venueCrowd(venue) : labels.crowd;
    const topTitle = venue && labels.venueTop ? labels.venueTop(venue) : labels.top;
    const crowdColor = index === 0 ? colors.crowd : (colors.crowdAlt ?? colors.crowd);
    const topColor = index === 0 ? colors.top : (colors.topAlt ?? colors.top);
    if (lines.crowd.length > 0) {
      out.push({
        id: `crowd-long${suffix}`,
        pane,
        kind: 'line',
        title: crowdTitle,
        color: crowdColor,
        data: lines.crowd,
      });
    }
    if (lines.top.length > 0) {
      out.push({
        id: `top-long${suffix}`,
        pane,
        kind: 'line',
        title: topTitle,
        color: topColor,
        data: lines.top,
      });
    }
  });
  return out;
}

export type RecordedFutures = { exchange: string; spanSec: number };

const RECORDED_KINDS = ['futures_oi', 'book', 'flow', 'liquidations'] as const;

export type RecordedPart = { kind: string; exchange: string; spanSec: number };

/** Stored series that have a span, in a stable kind order. */
export function recordedParts(span: HistorySpan | undefined): RecordedPart[] {
  const rows = span?.series ?? [];
  const out: RecordedPart[] = [];
  for (const kind of RECORDED_KINDS) {
    for (const row of rows) {
      if (row.kind !== kind || (row.spanSec ?? 0) <= 0 || !row.exchange) continue;
      out.push({ kind, exchange: String(row.exchange), spanSec: row.spanSec ?? 0 });
    }
  }
  return out;
}

/** Futures open-interest rows that have a stored span. */
export function futuresRecorded(span: HistorySpan | undefined): RecordedFutures[] {
  return recordedParts(span)
    .filter((row) => row.kind === 'futures_oi')
    .map((row) => ({ exchange: row.exchange, spanSec: row.spanSec }));
}

function rowsByVenue(items: FuturesPaneRow[] | undefined): Map<string, FuturesPaneRow[]> {
  const grouped = new Map<string, FuturesPaneRow[]>();
  for (const row of items ?? []) {
    const exchange = (row.exchange ?? '').toLowerCase();
    if (!exchange) continue;
    const list = grouped.get(exchange);
    if (list) list.push(row);
    else grouped.set(exchange, [row]);
  }
  return grouped;
}

function uniquePoints(points: ChartLinePoint[]): ChartLinePoint[] {
  const byTime = new Map<number, number>();
  for (const point of points) byTime.set(point.time, point.value);
  return [...byTime.entries()]
    .map(([time, value]) => ({ time, value }))
    .sort((a, b) => a.time - b.time);
}

/** USDT open interest, one line per venue that has samples. */
export function openInterestLines(
  items: FuturesPaneRow[] | undefined,
): { exchange: string; data: ChartLinePoint[] }[] {
  const grouped = rowsByVenue(items);
  const out: { exchange: string; data: ChartLinePoint[] }[] = [];
  for (const exchange of FUTURES_PANE_VENUES) {
    const data: ChartLinePoint[] = [];
    for (const row of grouped.get(exchange) ?? []) {
      const time = unixSeconds(row.sampledAt);
      const value = parseTapeNumber(row.value);
      if (time == null || value == null) continue;
      data.push({ time, value });
    }
    const points = uniquePoints(data);
    if (points.length > 0) out.push({ exchange, data: points });
  }
  return out;
}

/**
 * Funding percent, one line per venue. Settled prints win when that venue
 * has any. Predicted prints are used only when nothing has settled yet.
 */
export function fundingRateLines(
  items: FuturesPaneRow[] | undefined,
): { exchange: string; data: ChartLinePoint[] }[] {
  const grouped = rowsByVenue(items);
  const out: { exchange: string; data: ChartLinePoint[] }[] = [];
  for (const exchange of FUTURES_PANE_VENUES) {
    const rows = grouped.get(exchange) ?? [];
    const settled = rows.filter((row) => row.predicted === false);
    const source = settled.length > 0 ? settled : rows;
    const data: ChartLinePoint[] = [];
    for (const row of source) {
      const time = unixSeconds(row.sampledAt);
      const value = parseTapeNumber(row.ratePct);
      if (time == null || value == null) continue;
      data.push({ time, value });
    }
    const points = uniquePoints(data);
    if (points.length > 0) out.push({ exchange, data: points });
  }
  return out;
}

/**
 * Liquidation notional inside each candle. Positive is short liquidations.
 * Negative is long liquidations. A candle with both sides shows the difference.
 * Candles with no prints are omitted.
 */
export function liquidationNotionalByCandle(
  candles: { time: number }[],
  items: FuturesPaneRow[] | undefined,
  barDurationSec: number,
): ChartLinePoint[] {
  if (barDurationSec <= 0) return [];
  const events: { time: number; signed: number }[] = [];
  for (const row of items ?? []) {
    const time = unixSeconds(row.time ?? row.sampledAt);
    const notional = parseTapeNumber(row.notional);
    if (time == null || notional == null || notional === 0) continue;
    const side = (row.side ?? '').toLowerCase();
    const signed = side === 'long' ? -Math.abs(notional) : side === 'short' ? Math.abs(notional) : 0;
    if (signed === 0) continue;
    events.push({ time, signed });
  }
  events.sort((a, b) => a.time - b.time);
  const bars = candles
    .filter((candle) => Number.isFinite(candle.time))
    .sort((a, b) => a.time - b.time);
  const out: ChartLinePoint[] = [];
  let i = 0;
  for (const bar of bars) {
    const end = bar.time + barDurationSec;
    while (i < events.length && events[i]!.time < bar.time) i += 1;
    let j = i;
    let sum = 0;
    let any = false;
    while (j < events.length && events[j]!.time < end) {
      sum += events[j]!.signed;
      any = true;
      j += 1;
    }
    if (any && sum !== 0) out.push({ time: bar.time, value: sum });
    i = j;
  }
  return out;
}

/** Compact duration for the chart caption. */
export function formatRecordedSpan(sec: number): string {
  if (!Number.isFinite(sec) || sec <= 0) return '0m';
  if (sec < 3600) return `${Math.max(1, Math.round(sec / 60))}m`;
  if (sec < 48 * 3600) return `${Math.round(sec / 3600)}h`;
  const days = Math.floor(sec / 86400);
  const hours = Math.round((sec % 86400) / 3600);
  return hours > 0 ? `${days}d ${hours}h` : `${days}d`;
}
