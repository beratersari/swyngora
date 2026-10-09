import type {
  CandleChartFootprintCandle,
  CandleChartFootprintLevel,
  CandleChartFootprintRow,
  CandleChartHeatColumn,
  CandleChartLiqBar,
  CandleChartLiqCell,
  CandleChartLowerSeries,
  CandleChartOpenLevel,
} from '@/components/molecules/CandleChartHost';
import { pickOpenLevelBook } from '@/components/organisms/OpenLiquidationLevels/OpenLiquidationLevels.helpers';
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
  CandleFootprintBook,
  CandleFootprintReport,
  CandleFootprintRow,
  FlowAtPricesReport,
  HistorySpan,
  LiquidationHuntHeatmap,
  LiquidationOpenLevels,
  MarketLiquidationLevels,
  MarketLongShortRatio,
  LiquiditySweepReport,
  MarketBotTracker,
  OrderBookHeatmap,
  VolumeProfileReport,
} from '@/libs/api';
import { semanticColors, withAlpha } from '@/styles/tokens';
import { parseTapeNumber, unixSeconds } from './chartTape';

/** Pane order for Binance, Bybit, and OKX lines. */
export const FUTURES_PANE_VENUES = ['binance', 'bybit', 'okx'] as const;

/** Raw futures-history reads cap here. The chart sends `bucket` instead. */
export const FUTURES_PANE_LIMIT = 1000;

export type ChartPaneBucket = '5m' | '30m' | '2h' | '4h';

/** Stored chart window. Matches the futures archive cap. */
export const CHART_PANE_MAX_SEC = 90 * 24 * 60 * 60;

const PANE_BUCKET_SEC: Record<ChartPaneBucket, number> = {
  '5m': 5 * 60,
  '30m': 30 * 60,
  '2h': 2 * 60 * 60,
  '4h': 4 * 60 * 60,
};

/**
 * Step for stored open interest, funding, crowd, and liquidation notional.
 * Same cuts CVD uses: 5 minutes through 24 hours, 30 minutes through 7 days,
 * 2 hours through 30 days, and 4 hours after that.
 */
export function chartPaneBucket(spanSec: number): ChartPaneBucket {
  if (!Number.isFinite(spanSec) || spanSec <= DAY) return '5m';
  if (spanSec <= 7 * DAY) return '30m';
  if (spanSec <= 30 * DAY) return '2h';
  return '4h';
}

/**
 * Start of the stored window, floored to the step so a small tick does not refetch.
 * Flooring never starts before the 90-day cap. The next step is used when the
 * bucket open would fall outside that cap.
 */
export function chartPaneFromSec(firstSec: number, nowSec: number, bucket: ChartPaneBucket): number {
  const step = PANE_BUCKET_SEC[bucket];
  const earliest = nowSec - CHART_PANE_MAX_SEC;
  const from = Math.max(firstSec, earliest);
  const floored = Math.floor(from / step) * step;
  if (floored < earliest) return floored + step;
  return floored;
}

/** UTC day open and the current 4-hour open, in unix seconds. */
export function chartAnchors(nowSec: number): { day: number; block: number } {
  return {
    day: Math.floor(nowSec / DAY) * DAY,
    block: Math.floor(nowSec / (4 * HOUR)) * (4 * HOUR),
  };
}

/**
 * Change from the last sample at or before the anchor.
 * Empty when no sample has reached that open. The line starts at zero there.
 */
export function anchoredChange(points: ChartLinePoint[], anchorSec: number): ChartLinePoint[] {
  if (!Number.isFinite(anchorSec)) return [];
  const sorted = points
    .filter((point) => Number.isFinite(point.time) && Number.isFinite(point.value))
    .sort((a, b) => a.time - b.time);
  let baseline: number | null = null;
  for (const point of sorted) {
    if (point.time <= anchorSec) baseline = point.value;
    else break;
  }
  if (baseline == null) return [];
  const out: ChartLinePoint[] = [];
  for (const point of sorted) {
    if (point.time < anchorSec) continue;
    out.push({ time: point.time, value: point.value - baseline });
  }
  if (out.length > 0 && out[0]!.time > anchorSec) {
    out.unshift({ time: anchorSec, value: 0 });
  }
  return out;
}

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
  longPct?: string;
  sizeLongPct?: string;
  topAccLongPct?: string;
};

/** Day line uses the venue color. The 4-hour line uses a second color so the two anchors stay distinct. */
export function futuresAnchorColor(exchange: string, anchor: 'day' | '4h'): string {
  if (anchor === 'day') return futuresPaneColor(exchange);
  switch (exchange) {
    case 'bybit':
      return semanticColors.chart.down;
    case 'okx':
      return semanticColors.chart.up;
    default:
      return semanticColors.text.link;
  }
}

const HEAT_COLUMNS = 90;
const HEAT_LEVELS = 24;
const FOOTPRINT_ROWS = 48;
const LIQ_ROWS = 12;

const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Live order-book tape. A longer chart view uses stored minute ladders. */
export const CHART_BOOK_LIVE_MAX = 30 * MINUTE;

/**
 * Heatmap lookback, in seconds, for the candles on screen.
 * Above 30 minutes the API reads stored ladders, capped at 30 days.
 */
export function chartBookWindow(spanSec: number): number {
  if (!Number.isFinite(spanSec) || spanSec <= 5 * MINUTE) return 5 * MINUTE;
  if (spanSec <= 10 * MINUTE) return 10 * MINUTE;
  if (spanSec <= 15 * MINUTE) return 15 * MINUTE;
  if (spanSec <= CHART_BOOK_LIVE_MAX) return CHART_BOOK_LIVE_MAX;
  if (spanSec <= HOUR) return HOUR;
  if (spanSec <= 4 * HOUR) return 4 * HOUR;
  if (spanSec <= DAY) return DAY;
  if (spanSec <= 3 * DAY) return 3 * DAY;
  if (spanSec <= 7 * DAY) return 7 * DAY;
  return 30 * DAY;
}

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

const PERP_BOOK_RINGS = [
  { pct: 1, inner: 0.5 },
  { pct: 0.5, inner: 0.1 },
  { pct: 0.1, inner: 0 },
] as const;

type PerpBookBand = { rangePct?: string; bid?: string; ask?: string };
type PerpBookColumn = { time?: string; mid?: string; coverBid?: string; coverAsk?: string; bands?: PerpBookBand[] };

/** Perp bid and ask bands drawn behind the candles. Nested rings avoid painting the same size twice. */
export function perpBookCells(report?: { stepSec?: number; columns?: PerpBookColumn[] } | null): CandleChartLiqCell[] {
  const columns = report?.columns ?? [];
  if (columns.length === 0) return [];
  const stepSec = report?.stepSec && report.stepSec > 0 ? report.stepSec : 60;
  const rings: { time: number; priceHi: number; priceLo: number; notional: number; side: 'bid' | 'ask' }[] = [];
  for (const col of columns) {
    const time = unixSeconds(col.time);
    const mid = parseTapeNumber(col.mid);
    const coverBid = parseTapeNumber(col.coverBid) ?? 0;
    const coverAsk = parseTapeNumber(col.coverAsk) ?? 0;
    if (time == null || mid == null || mid <= 0) continue;
    const byPct = new Map<number, { bid: number; ask: number }>();
    for (const band of col.bands ?? []) {
      const pct = parseTapeNumber(band.rangePct);
      const bid = parseTapeNumber(band.bid) ?? 0;
      const ask = parseTapeNumber(band.ask) ?? 0;
      if (pct == null || bid + ask <= 0) continue;
      byPct.set(pct, { bid, ask });
    }
    const tighter = (pct: number, side: 'bid' | 'ask') => {
      let best = 0;
      for (const [bandPct, band] of byPct) {
        if (bandPct < pct) best = Math.max(best, side === 'bid' ? band.bid : band.ask);
      }
      return best;
    };
    for (const ring of PERP_BOOK_RINGS) {
      const outer = byPct.get(ring.pct);
      if (!outer) continue;
      const bid = Math.max(0, outer.bid - tighter(ring.pct, 'bid'));
      const ask = Math.max(0, outer.ask - tighter(ring.pct, 'ask'));
      const bidEdge = coverBid > 0 ? Math.min(ring.pct, coverBid) : ring.pct;
      const askEdge = coverAsk > 0 ? Math.min(ring.pct, coverAsk) : ring.pct;
      const priceLo = mid * (1 - bidEdge / 100);
      const priceHi = mid * (1 + askEdge / 100);
      const innerLo = mid * (1 - ring.inner / 100);
      const innerHi = mid * (1 + ring.inner / 100);
      if (bid > 0 && bidEdge > ring.inner) rings.push({ time, priceHi: innerLo, priceLo, notional: bid, side: 'bid' });
      if (ask > 0 && askEdge > ring.inner) rings.push({ time, priceHi, priceLo: innerHi, notional: ask, side: 'ask' });
    }
  }
  let max = 0;
  for (const ring of rings) max = Math.max(max, ring.notional);
  if (max <= 0) return [];
  return rings.map((ring) => ({
    time: ring.time,
    stepSec,
    priceHi: ring.priceHi,
    priceLo: ring.priceLo,
    color: withAlpha(
      ring.side === 'bid' ? semanticColors.chart.up : semanticColors.chart.down,
      0.18 + 0.55 * Math.min(1, ring.notional / max),
    ),
  }));
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

/** Newest candles requested before the chart reports a visible span. */
export const CANDLE_FOOTPRINT_DEFAULT_CANDLES = 120;

const CANDLE_FOOTPRINT_MAX_CANDLES = 400;
const CANDLE_FOOTPRINT_QUANTUM = 8;

/**
 * Visible candle range snapped to candle opens, then to 8-candle blocks,
 * and capped at 400 candles. A one-candle pan stays on the same block.
 */
export function candleFootprintSpan(
  intervalSec: number,
  fromSec: number,
  toSec: number,
): { from: number; to: number } | null {
  if (!Number.isFinite(intervalSec) || intervalSec <= 0) return null;
  if (!Number.isFinite(fromSec) || !Number.isFinite(toSec) || toSec <= fromSec) return null;
  const step = intervalSec;
  let from = Math.floor(fromSec / step) * step;
  let to = Math.ceil(toSec / step) * step;
  if (to <= from) to = from + step;
  const maxSpan = CANDLE_FOOTPRINT_MAX_CANDLES * step;
  if (to - from > maxSpan) from = to - maxSpan;
  const quantum = step * CANDLE_FOOTPRINT_QUANTUM;
  from = Math.floor(from / quantum) * quantum;
  to = Math.ceil(to / quantum) * quantum;
  if (to - from > maxSpan) {
    to = Math.ceil(toSec / quantum) * quantum;
    from = to - maxSpan;
  }
  if (!(to > from)) return null;
  return { from, to };
}

export type CandleFootprintChart = {
  candles: CandleChartFootprintCandle[];
  stepSec: number;
  definite: boolean;
  truncated: boolean;
  missing: string[];
  /** Venue drawn when the combined book is withheld. Empty when combined is drawn. */
  exchange: string;
};

const EMPTY_CANDLE_FOOTPRINT: CandleFootprintChart = {
  candles: [],
  stepSec: 0,
  definite: false,
  truncated: false,
  missing: [],
  exchange: '',
};

function candleFootprintRows(rows: CandleFootprintRow[] | undefined): CandleChartFootprintRow[] {
  const out: CandleChartFootprintRow[] = [];
  for (const row of rows ?? []) {
    const price = parseTapeNumber(row.price);
    const high = parseTapeNumber(row.high);
    const buy = parseTapeNumber(row.buyVolume) ?? 0;
    const sell = parseTapeNumber(row.sellVolume) ?? 0;
    if (
      price == null ||
      high == null ||
      !Number.isFinite(price) ||
      !Number.isFinite(high) ||
      buy + sell <= 0
    ) {
      continue;
    }
    const imbalance = row.imbalance === 'buy' || row.imbalance === 'sell' ? row.imbalance : '';
    out.push({ price, high, buy, sell, poc: row.isPoc === true, imbalance });
  }
  return out.sort((a, b) => a.price - b.price);
}

function candleFootprintCandles(book: CandleFootprintBook | undefined): CandleChartFootprintCandle[] {
  const out: CandleChartFootprintCandle[] = [];
  for (const candle of book?.candles ?? []) {
    const time = unixSeconds(candle.time);
    if (time == null) continue;
    const rows = candleFootprintRows(candle.rows);
    if (rows.length === 0) continue;
    out.push({ time, rows });
  }
  return out;
}

/**
 * Combined rows when every venue was recorded and the book has candles.
 * Otherwise the page's venue, then any other venue that has candles.
 */
export function candleFootprintView(
  report: CandleFootprintReport | undefined,
  pageExchange: string,
): CandleFootprintChart {
  if (!report) return EMPTY_CANDLE_FOOTPRINT;
  const missing = (report.missing ?? []).filter((id): id is string => Boolean(id));
  const base: CandleFootprintChart = {
    candles: [],
    stepSec: report.stepSec != null && report.stepSec > 0 ? report.stepSec : 0,
    definite: report.definite === true,
    truncated: report.truncated === true,
    missing,
    exchange: '',
  };
  if (base.definite) {
    const combined = candleFootprintCandles(report.combined);
    if (combined.length > 0) return { ...base, candles: combined };
  }
  const want = pageExchange.toLowerCase();
  const venues = report.venues ?? [];
  const page = venues.find((venue) => (venue.exchange ?? '').toLowerCase() === want);
  const pageCandles = candleFootprintCandles(page);
  if (pageCandles.length > 0) return { ...base, candles: pageCandles, exchange: want };
  for (const venue of venues) {
    const candles = candleFootprintCandles(venue);
    if (candles.length > 0) {
      return { ...base, candles, exchange: (venue.exchange ?? '').toLowerCase() };
    }
  }
  return base;
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

export type OpenLevelNotes = {
  mix: string;
  missingVenues: string[];
  /** True only when the selected book came back and is not a definite combined read. */
  withheld: boolean;
};

/**
 * Bands drawn on the candles. Closed inventory changes stay off the canvas.
 * A withheld combined book draws nothing.
 */
export function openLevelSegments(
  data: LiquidationOpenLevels | undefined,
  venue: LiqHeatVenue,
): CandleChartOpenLevel[] {
  const book = pickOpenLevelBook(data, venue);
  if (!book || book.definite === false) return [];
  const out: CandleChartOpenLevel[] = [];
  for (const segment of book.segments ?? []) {
    if (segment.status !== 'open' && segment.status !== 'hit') continue;
    if (segment.side !== 'long' && segment.side !== 'short') continue;
    const openedAt = unixSeconds(segment.openedAt);
    if (
      segment.price == null ||
      segment.notional == null ||
      openedAt == null ||
      !Number.isFinite(segment.price) ||
      segment.price <= 0 ||
      !Number.isFinite(segment.notional) ||
      segment.notional <= 0
    ) {
      continue;
    }
    // Year 1 is an unset close. Keep it off the band so the line is not drawn backward.
    const closedAt = unixSeconds(segment.closedAt);
    out.push({
      side: segment.side,
      price: segment.price,
      notional: segment.notional,
      openedAt,
      ...(closedAt != null && closedAt >= openedAt ? { closedAt } : {}),
      status: segment.status,
    });
  }
  return out;
}

/** Uncapped open long size minus open short size. Empty when the book is withheld. */
export function openLevelDelta(
  data: LiquidationOpenLevels | undefined,
  venue: LiqHeatVenue,
): ChartLinePoint[] {
  const book = pickOpenLevelBook(data, venue);
  if (!book || book.definite === false) return [];
  const out: ChartLinePoint[] = [];
  for (const point of book.cumulative ?? []) {
    const time = unixSeconds(point.time);
    if (time == null || point.delta == null || !Number.isFinite(point.delta)) continue;
    out.push({ time, value: point.delta });
  }
  out.sort((a, b) => a.time - b.time);
  return out;
}

/** Mix, missing venues, and whether this venue's book was withheld. */
export function openLevelNotes(
  data: LiquidationOpenLevels | undefined,
  venue: LiqHeatVenue,
): OpenLevelNotes {
  const book = pickOpenLevelBook(data, venue);
  return {
    mix: formatLeverageMix(data?.leverageMix),
    missingVenues: (data?.missingVenues ?? []).filter((id) => id.length > 0),
    withheld: book != null && book.definite === false,
  };
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

/**
 * Liquidation notional already summed into chart steps.
 * Positive is short liquidations. Negative is long liquidations.
 * Venues at the same step are added. A step with no prints is omitted.
 */
export function liquidationBucketBars(items: FuturesPaneRow[] | undefined): ChartLinePoint[] {
  const byTime = new Map<number, number>();
  for (const row of items ?? []) {
    const time = unixSeconds(row.time ?? row.sampledAt);
    const notional = parseTapeNumber(row.notional);
    if (time == null || notional == null || notional === 0) continue;
    const side = (row.side ?? '').toLowerCase();
    const signed = side === 'long' ? -Math.abs(notional) : side === 'short' ? Math.abs(notional) : 0;
    if (signed === 0) continue;
    byTime.set(time, (byTime.get(time) ?? 0) + signed);
  }
  return [...byTime.entries()]
    .filter(([, value]) => value !== 0)
    .map(([time, value]) => ({ time, value }))
    .sort((a, b) => a.time - b.time);
}

export type CrowdStored = {
  exchange: string;
  crowd: ChartLinePoint[];
  top: ChartLinePoint[];
};

/** Crowd long percent and top-trader long percent from stored long/short rows. */
export function crowdStoredLines(items: FuturesPaneRow[] | undefined): CrowdStored[] {
  const grouped = rowsByVenue(items);
  const out: CrowdStored[] = [];
  for (const exchange of FUTURES_PANE_VENUES) {
    const crowd: ChartLinePoint[] = [];
    const top: ChartLinePoint[] = [];
    for (const row of grouped.get(exchange) ?? []) {
      const time = unixSeconds(row.sampledAt);
      const crowdPct = parseTapeNumber(row.longPct);
      if (time == null || crowdPct == null) continue;
      crowd.push({ time, value: crowdPct });
      const sizePct = parseTapeNumber(row.sizeLongPct);
      const accountPct = parseTapeNumber(row.topAccLongPct);
      const topPct = sizePct != null && sizePct > 0 ? sizePct : accountPct;
      if (topPct != null && topPct > 0) top.push({ time, value: topPct });
    }
    const crowdPoints = uniquePoints(crowd);
    const topPoints = uniquePoints(top);
    if (crowdPoints.length > 0 || topPoints.length > 0) {
      out.push({ exchange, crowd: crowdPoints, top: topPoints });
    }
  }
  return out;
}

/** Top long percent minus crowd long percent, only where both exist at the same time. */
export function crowdGapLines(rows: CrowdStored[]): { exchange: string; data: ChartLinePoint[] }[] {
  const out: { exchange: string; data: ChartLinePoint[] }[] = [];
  for (const row of rows) {
    const top = new Map(row.top.map((point) => [point.time, point.value]));
    const data: ChartLinePoint[] = [];
    for (const point of row.crowd) {
      const topPct = top.get(point.time);
      if (topPct == null) continue;
      data.push({ time: point.time, value: topPct - point.value });
    }
    if (data.length > 0) out.push({ exchange: row.exchange, data });
  }
  return out;
}

/**
 * Bybit never publishes a top-trader ratio. Another venue is named only when
 * its crowd line is present and its top-trader share is not.
 */
export function venuesWithoutTop(rows: CrowdStored[]): string[] {
  const hasTop = new Set(rows.filter((row) => row.top.length > 0).map((row) => row.exchange));
  return FUTURES_PANE_VENUES.filter((id) => {
    if (hasTop.has(id)) return false;
    if (id === 'bybit') return true;
    return rows.some((row) => row.exchange === id && row.crowd.length > 0);
  });
}

export function crowdStoredSeries(
  rows: CrowdStored[],
  labels: CrowdChartLabels,
  colors: { crowd: string; top: string; crowdAlt?: string; topAlt?: string },
  pane: number,
): CandleChartLowerSeries[] {
  const out: CandleChartLowerSeries[] = [];
  rows.forEach((row, index) => {
    const suffix = rows.length > 1 ? `-${row.exchange}` : '';
    const crowdTitle = labels.venueCrowd ? labels.venueCrowd(row.exchange) : labels.crowd;
    const topTitle = labels.venueTop ? labels.venueTop(row.exchange) : labels.top;
    const crowdColor = index === 0 ? colors.crowd : (colors.crowdAlt ?? colors.crowd);
    const topColor = index === 0 ? colors.top : (colors.topAlt ?? colors.top);
    if (row.crowd.length > 0) {
      out.push({
        id: `crowd-long${suffix}`,
        pane,
        kind: 'line',
        title: crowdTitle,
        color: crowdColor,
        data: row.crowd,
      });
    }
    if (row.top.length > 0) {
      out.push({
        id: `top-long${suffix}`,
        pane,
        kind: 'line',
        title: topTitle,
        color: topColor,
        data: row.top,
      });
    }
  });
  return out;
}

export type AnchoredVenueSeries = {
  series: CandleChartLowerSeries[];
  missingDay: string[];
  missingBlock: string[];
};

/** Change since the UTC day open and since the current 4-hour open, one pair of lines per venue. */
export function anchoredVenueSeries(
  lines: { exchange: string; data: ChartLinePoint[] }[],
  anchors: { day: number; block: number },
  titles: { day: (exchange: string) => string; block: (exchange: string) => string },
  pane: number,
  idPrefix: string,
): AnchoredVenueSeries {
  const series: CandleChartLowerSeries[] = [];
  const missingDay: string[] = [];
  const missingBlock: string[] = [];
  for (const line of lines) {
    if (line.data.length === 0) continue;
    const day = anchoredChange(line.data, anchors.day);
    const block = anchoredChange(line.data, anchors.block);
    if (day.length === 0) missingDay.push(line.exchange);
    else {
      series.push({
        id: `${idPrefix}-day-${line.exchange}`,
        pane,
        kind: 'line',
        title: titles.day(line.exchange),
        color: futuresAnchorColor(line.exchange, 'day'),
        data: day,
      });
    }
    if (block.length === 0) missingBlock.push(line.exchange);
    else {
      series.push({
        id: `${idPrefix}-4h-${line.exchange}`,
        pane,
        kind: 'line',
        title: titles.block(line.exchange),
        color: futuresAnchorColor(line.exchange, '4h'),
        data: block,
      });
    }
  }
  return { series, missingDay, missingBlock };
}

/** Click-anchor lines from the positioning-anchor read. Empty points stay off. */
export function positioningClickLines(
  series: { exchange?: string; points?: { time?: string; value?: string }[] }[] | undefined,
  title: (exchange: string) => string,
  pane: number,
  idPrefix: string,
): CandleChartLowerSeries[] {
  const out: CandleChartLowerSeries[] = [];
  for (const row of series ?? []) {
    const exchange = (row.exchange ?? '').toLowerCase();
    const data: ChartLinePoint[] = [];
    for (const point of row.points ?? []) {
      const time = unixSeconds(point.time);
      const value = parseTapeNumber(point.value);
      if (time == null || value == null) continue;
      data.push({ time, value });
    }
    const points = uniquePoints(data);
    if (!exchange || points.length === 0) continue;
    out.push({
      id: `${idPrefix}-${exchange}`,
      pane,
      kind: 'line',
      title: title(exchange),
      color: futuresPaneColor(exchange),
      data: points,
    });
  }
  return out;
}

const TAPE_MAX_SEC = 30 * DAY;
const TAPE_QUANTUM_SEC = 15 * MINUTE;
/** Spot sweeps are the last ~7 days of 15-minute candles. */
export const SWEEP_LOOKBACK_SEC = 7 * DAY;

export type ChartTapeSpan = { from: number; to: number };

/**
 * Visible candle range for the profile request.
 * Rounded to 15 minutes and capped at 30 days so a small pan does not refetch.
 */
export function chartTapeSpan(intervalSec: number, fromSec: number, toSec: number): ChartTapeSpan | null {
  if (!Number.isFinite(intervalSec) || intervalSec <= 0) return null;
  if (!Number.isFinite(fromSec) || !Number.isFinite(toSec) || toSec <= fromSec) return null;
  let from = fromSec;
  let to = toSec;
  if (to - from > TAPE_MAX_SEC) from = to - TAPE_MAX_SEC;
  const quantum = Math.max(intervalSec, TAPE_QUANTUM_SEC);
  from = Math.floor(from / quantum) * quantum;
  to = Math.ceil(to / quantum) * quantum;
  if (to - from > TAPE_MAX_SEC) from = to - TAPE_MAX_SEC;
  if (!(to > from)) return null;
  return { from, to };
}

export type ChartProfileLine = {
  id: string;
  price: number;
  role: 'poc' | 'vaHigh' | 'vaLow';
};

export type ChartProfileRead = {
  lines: ChartProfileLine[];
  /** `combined` or the single venue drawn. */
  exchange: string;
  missing: string[];
  definite: boolean;
};

function profilePrice(raw: string | undefined): number | null {
  const n = parseTapeNumber(raw);
  return n != null && n > 0 ? n : null;
}

function samePrice(a: number, b: number): boolean {
  const scale = Math.max(Math.abs(a), Math.abs(b), 1);
  return Math.abs(a - b) / scale < 1e-8;
}

/** POC and value-area edges. Combined only when every venue reaches the window open. */
export function chartProfileRead(
  report: VolumeProfileReport | null | undefined,
  pageExchange: string,
): ChartProfileRead {
  const missing = report?.missing ?? [];
  const empty: ChartProfileRead = { lines: [], exchange: '', missing, definite: false };
  if (!report) return empty;
  const page = pageExchange.toLowerCase();
  let venue = report.venues?.find((row) => (row.exchange ?? '').toLowerCase() === page && !row.error);
  let exchange = page;
  if (report.definite && report.combined && !report.combined.error) {
    venue = report.combined;
    exchange = 'combined';
  } else if (!venue) {
    venue = report.venues?.find((row) => !row.error && (row.poc?.price || row.valueArea?.low));
    exchange = (venue?.exchange ?? '').toLowerCase();
  }
  if (!venue) return empty;
  const pocLow = profilePrice(venue.poc?.price);
  const pocHigh = profilePrice(venue.poc?.high);
  const poc = pocLow != null && pocHigh != null ? (pocLow + pocHigh) / 2 : pocLow;
  const vaLow = profilePrice(venue.valueArea?.low);
  const vaHigh = profilePrice(venue.valueArea?.high);
  const lines: ChartProfileLine[] = [];
  if (poc != null) lines.push({ id: 'profile-poc', price: poc, role: 'poc' });
  if (vaHigh != null && (poc == null || !samePrice(vaHigh, poc))) {
    lines.push({ id: 'profile-va-high', price: vaHigh, role: 'vaHigh' });
  }
  if (vaLow != null && (poc == null || !samePrice(vaLow, poc)) && (vaHigh == null || !samePrice(vaLow, vaHigh))) {
    lines.push({ id: 'profile-va-low', price: vaLow, role: 'vaLow' });
  }
  return {
    lines,
    exchange,
    missing: exchange === 'combined' ? [] : missing,
    definite: Boolean(report.definite) && exchange === 'combined',
  };
}

export type ChartTapeMark = {
  time: number;
  kind: 'sweep-high' | 'sweep-low' | 'bot-buy' | 'bot-sell';
  count?: number;
};

function overlapsSpan(start: number, end: number, span: ChartTapeSpan | null): boolean {
  if (!span) return true;
  return end >= span.from && start <= span.to;
}

/** Page venue when it has a spot sweep book. Coinbase uses Binance. */
export function chartSweepVenue(pageExchange: string): 'binance' | 'bybit' {
  return pageExchange.toLowerCase() === 'bybit' ? 'bybit' : 'binance';
}

/** Sweep pokes whose pierce time sits in the visible span. */
export function chartSweepMarks(
  report: LiquiditySweepReport | null | undefined,
  venue: string,
  span: ChartTapeSpan | null,
): ChartTapeMark[] {
  const want = venue.toLowerCase();
  const rows = report?.venues?.find((row) => (row.exchange ?? '').toLowerCase() === want)?.sweeps ?? [];
  const out: ChartTapeMark[] = [];
  for (const row of rows) {
    const time = unixSeconds(row.piercedAt);
    if (time == null || !overlapsSpan(time, time, span)) continue;
    const side = (row.side ?? '').toLowerCase();
    if (side !== 'high' && side !== 'low') continue;
    out.push({ time, kind: side === 'high' ? 'sweep-high' : 'sweep-low' });
  }
  return out.sort((a, b) => a.time - b.time);
}

/** True when the visible range ends before the sweep lookback. */
export function sweepSpanOutside(span: ChartTapeSpan | null, nowSec: number): boolean {
  if (!span || !Number.isFinite(nowSec)) return false;
  return span.to < nowSec - SWEEP_LOOKBACK_SEC;
}

/** Repeated clips whose burst overlaps the visible span. */
export function chartBotMarks(
  report: MarketBotTracker | null | undefined,
  span: ChartTapeSpan | null,
): ChartTapeMark[] {
  const out: ChartTapeMark[] = [];
  for (const cluster of report?.clusters ?? []) {
    const start = unixSeconds(cluster.from);
    const end = unixSeconds(cluster.to) ?? start;
    if (start == null || end == null || !overlapsSpan(start, end, span)) continue;
    const side = (cluster.side ?? '').toLowerCase();
    if (side !== 'buy' && side !== 'sell') continue;
    out.push({
      time: start,
      kind: side === 'buy' ? 'bot-buy' : 'bot-sell',
      count: cluster.count ?? 0,
    });
  }
  return out.sort((a, b) => a.time - b.time);
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
