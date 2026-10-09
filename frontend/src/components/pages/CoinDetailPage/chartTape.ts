import type { MarketCvd, MarketLiquidationLevels } from '@/libs/api';

export type TapeDeltaPoint = {
  time: number;
  delta: number;
  /** Running sum from the start of the returned range. */
  cvd?: number | null;
};

export type ChartBarPoint = {
  time: number;
  value: number;
};

export type ChartLiqLevel = {
  id: string;
  price: number;
  side: 'long' | 'short';
  leverages: number[];
};

const LIQ_LEVERAGES = [100, 50, 25, 10] as const;

/** Matches backend cvdBarPriceFlatPct. A smaller move does not close a run. */
const CVD_BAR_FLAT_PCT = 0.05;

export type CvdChartSplit = {
  time: number;
  kind: 'price_up_cvd_down' | 'price_down_cvd_up';
  duration: string;
  durationSeconds: number;
};

function chartSplit(venue: {
  divergence?: { kind?: string; since?: string; lastAt?: string; duration?: string; durationSeconds?: number };
  points?: { time?: string; delta?: string; priceChangePct?: string }[];
} | undefined): CvdChartSplit | null {
  const kind = venue?.divergence?.kind;
  if (kind !== 'price_up_cvd_down' && kind !== 'price_down_cvd_up') return null;
  const time = unixSeconds(venue?.divergence?.since);
  const durationSeconds = venue?.divergence?.durationSeconds ?? 0;
  if (time == null || durationSeconds <= 0) return null;
  const lastAt = unixSeconds(venue?.divergence?.lastAt) ?? time + durationSeconds;
  for (const point of venue?.points ?? []) {
    const at = unixSeconds(point.time);
    if (at == null || at <= lastAt) continue;
    if (!cvdPointQuiet(point)) return null;
  }
  return {
    time,
    kind,
    duration: venue?.divergence?.duration ?? '',
    durationSeconds,
  };
}

/** A flat price or a zero delta does not end the run. An agreeing bar does. */
function cvdPointQuiet(point: { delta?: string; priceChangePct?: string }): boolean {
  const pct = parseTapeNumber(point.priceChangePct);
  const delta = parseTapeNumber(point.delta);
  if (pct == null || delta == null) return false;
  return Math.abs(pct) < CVD_BAR_FLAT_PCT || delta === 0;
}

/** Parse a signed API quantity (`+1.5`, `-2`, `0`). */
export function parseTapeNumber(raw: string | number | null | undefined): number | null {
  if (typeof raw === 'number') return Number.isFinite(raw) ? raw : null;
  if (!raw) return null;
  const n = Number(String(raw).replace(/,/g, '').replace(/^\+/, '').trim());
  return Number.isFinite(n) ? n : null;
}

export function unixSeconds(iso: string | undefined): number | null {
  if (!iso) return null;
  const ms = Date.parse(iso);
  if (!Number.isFinite(ms)) return null;
  return Math.floor(ms / 1000);
}

/** UTC day open (00:00) containing `atSec`. */
export function utcDayOpenSec(atSec: number): number {
  const d = new Date(atSec * 1000);
  return Math.floor(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()) / 1000);
}

/** Current 4-hour clock open (00/04/08/12/16/20 UTC) containing `atSec`. */
export function utc4hOpenSec(atSec: number): number {
  const d = new Date(atSec * 1000);
  const hour = Math.floor(d.getUTCHours() / 4) * 4;
  return Math.floor(
    Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate(), hour) / 1000,
  );
}

function seriesStart(points: { time?: string }[] | undefined): number | null {
  let min: number | null = null;
  for (const point of points ?? []) {
    const time = unixSeconds(point.time);
    if (time == null) continue;
    if (min == null || time < min) min = time;
  }
  return min;
}

/** Futures combined needs every one of these. A missing venue is not zero. */
export const FUTURES_CVD_VENUES = ['binance', 'bybit', 'okx'] as const;
/** Spot CVD is Binance klines and Bybit trades. OKX has no spot tape. */
export const SPOT_CVD_VENUES = ['binance', 'bybit'] as const;

export type CvdChartSeries = {
  points: TapeDeltaPoint[];
  dayFrom: number | null;
  blockFrom: number | null;
  complete: boolean;
  range: string;
  bar: string;
  source: string;
  withheld: string[];
  /** Current run on the venue whose line is drawn. Absent when price and CVD agree. */
  split: CvdChartSplit | null;
};

type CvdSide = {
  venues?: MarketCvd['venues'];
  combined?: MarketCvd['combined'];
};

function venueId(venue: { exchange?: string } | undefined): string {
  return (venue?.exchange ?? '').trim().toLowerCase();
}

/** Ready means this venue's own tape reaches the window and has bars. */
function venueReady(venue: NonNullable<MarketCvd['venues']>[number] | undefined): boolean {
  if (!venue || venue.error) return false;
  if (venue.complete !== true) return false;
  return (venue.points?.length ?? 0) > 0;
}

/**
 * CVD for one market.
 * Combined is drawn only when it is complete, it reaches the page venue,
 * and every expected venue's own tape is complete. Otherwise the page venue
 * is drawn. A Coinbase or OKX page asks for Binance spot. Venues that do
 * not cover the window are named.
 */
function pickCvdSeries(
  side: CvdSide | undefined,
  pageExchange: string,
  expected: readonly string[],
  range: string,
  bar: string,
): CvdChartSeries {
  const venues = side?.venues ?? [];
  const want = pageExchange.trim().toLowerCase();
  const page = venues.find((venue) => venueId(venue) === want && (venue.points?.length ?? 0) > 0);
  const combined =
    side?.combined?.points && side.combined.points.length > 0 ? side.combined : undefined;
  const pageStart = seriesStart(page?.points);
  const combinedStart = seriesStart(combined?.points);
  const everyVenueReady = expected.every((id) => venues.some((venue) => venueId(venue) === id && venueReady(venue)));
  const combinedCovers =
    everyVenueReady &&
    combined != null &&
    combined.complete === true &&
    (pageStart == null || (combinedStart != null && combinedStart <= pageStart + 60));
  const venue = combinedCovers
    ? combined
    : (page ?? venues.find((item) => (item.points?.length ?? 0) > 0));
  const source = combinedCovers ? 'combined' : venueId(venue);
  const drawnStart = seriesStart(venue?.points);
  const withheld =
    source === 'combined' || source === ''
      ? []
      : expected.filter((id) => {
          const row = venues.find((item) => venueId(item) === id);
          if (id === source && venueReady(row)) return false;
          if (!venueReady(row)) return true;
          const start = seriesStart(row?.points);
          return start == null || (drawnStart != null && start > drawnStart + 60);
        });
  const points: TapeDeltaPoint[] = [];
  for (const p of venue?.points ?? []) {
    const time = unixSeconds(p.time);
    const delta = parseTapeNumber(p.delta);
    if (time == null || delta == null) continue;
    points.push({ time, delta, cvd: parseTapeNumber(p.cvd) });
  }
  points.sort((a, b) => a.time - b.time);
  const last = points.length > 0 ? points[points.length - 1]!.time : null;
  const dayAnchor = unixSeconds(venue?.anchors?.find((a) => a.anchor === 'day')?.from);
  const blockAnchor = unixSeconds(venue?.anchors?.find((a) => a.anchor === '4h')?.from);
  return {
    points,
    dayFrom: dayAnchor ?? (last != null ? utcDayOpenSec(last) : null),
    blockFrom: blockAnchor ?? (last != null ? utc4hOpenSec(last) : null),
    complete: venue?.complete === true,
    range,
    bar,
    source,
    withheld,
    split: chartSplit(venue),
  };
}

/** Futures CVD. A spot page passes Binance as the page venue. */
export function futuresCvdSeries(report: MarketCvd | undefined, pageExchange = 'binance'): CvdChartSeries {
  return pickCvdSeries(
    { venues: report?.venues, combined: report?.combined },
    pageExchange,
    FUTURES_CVD_VENUES,
    report?.range ?? '24h',
    report?.bar ?? '5m',
  );
}

/**
 * Spot CVD. OKX is not expected. A Coinbase or OKX page passes Binance so the
 * line is named Binance when combined is off.
 */
export function spotCvdSeries(report: MarketCvd | undefined, pageExchange = 'binance'): CvdChartSeries {
  return pickCvdSeries(
    { venues: report?.spotVenues, combined: report?.spotCombined },
    pageExchange,
    SPOT_CVD_VENUES,
    report?.range ?? '24h',
    report?.bar ?? '5m',
  );
}

/**
 * Sum aggressive buy−sell into each candle. A 5-minute print belongs to the
 * candle whose open contains that print. Candles with no prints are omitted.
 */
export function deltaByCandle(
  candles: { time: number }[],
  points: TapeDeltaPoint[],
  barDurationSec: number,
): ChartBarPoint[] {
  if (barDurationSec <= 0) return [];
  const pts = points
    .filter((p) => Number.isFinite(p.time) && Number.isFinite(p.delta))
    .sort((a, b) => a.time - b.time);
  const bars = candles
    .filter((c) => Number.isFinite(c.time))
    .sort((a, b) => a.time - b.time);
  const out: ChartBarPoint[] = [];
  let i = 0;
  for (const bar of bars) {
    const end = bar.time + barDurationSec;
    while (i < pts.length && pts[i]!.time < bar.time) i += 1;
    let j = i;
    let sum = 0;
    let any = false;
    while (j < pts.length && pts[j]!.time < end) {
      sum += pts[j]!.delta;
      any = true;
      j += 1;
    }
    if (any) out.push({ time: bar.time, value: sum });
    i = j;
  }
  return out;
}

/**
 * Running CVD from the start of the returned range, sampled on candle opens.
 * A candle keeps the last sum so the line stays continuous between coarser bars.
 */
export function runningCvdByCandle(
  candles: { time: number }[],
  points: TapeDeltaPoint[],
  barDurationSec: number,
): ChartBarPoint[] {
  if (barDurationSec <= 0) return [];
  const pts = points
    .filter((p) => Number.isFinite(p.time) && p.cvd != null && Number.isFinite(p.cvd))
    .sort((a, b) => a.time - b.time);
  if (pts.length === 0) return [];
  const bars = candles
    .filter((c) => Number.isFinite(c.time))
    .sort((a, b) => a.time - b.time);
  const out: ChartBarPoint[] = [];
  let i = 0;
  let last: number | null = null;
  const first = pts[0]!.time;
  for (const bar of bars) {
    const end = bar.time + barDurationSec;
    while (i < pts.length && pts[i]!.time < end) {
      last = pts[i]!.cvd ?? last;
      i += 1;
    }
    if (last != null && bar.time + barDurationSec > first) {
      out.push({ time: bar.time, value: last });
    }
  }
  return out;
}

/**
 * Running buy−sell reset to zero at `anchorFromSec`, sampled on candle opens.
 * Later candles keep the last sum so the line stays continuous between prints.
 */
export function anchoredCvdByCandle(
  candles: { time: number }[],
  points: TapeDeltaPoint[],
  anchorFromSec: number,
  barDurationSec: number,
): ChartBarPoint[] {
  if (!Number.isFinite(anchorFromSec) || barDurationSec <= 0) return [];
  const pts = points
    .filter((p) => Number.isFinite(p.time) && Number.isFinite(p.delta) && p.time >= anchorFromSec)
    .sort((a, b) => a.time - b.time);
  const bars = candles
    .filter((c) => Number.isFinite(c.time) && c.time + barDurationSec > anchorFromSec)
    .sort((a, b) => a.time - b.time);
  const out: ChartBarPoint[] = [];
  let i = 0;
  let sum = 0;
  let started = false;
  for (const bar of bars) {
    const end = bar.time + barDurationSec;
    while (i < pts.length && pts[i]!.time < end) {
      sum += pts[i]!.delta;
      started = true;
      i += 1;
    }
    if (started) out.push({ time: bar.time, value: sum });
  }
  return out;
}

export const CVD_FETCH_RANGES = ['24h', '7d', '30d', '90d'] as const;
export type CvdFetchRange = (typeof CVD_FETCH_RANGES)[number];

const CVD_RANGE_SEC: Record<CvdFetchRange, number> = {
  '24h': 24 * 60 * 60,
  '7d': 7 * 24 * 60 * 60,
  '30d': 30 * 24 * 60 * 60,
  '90d': 90 * 24 * 60 * 60,
};

/** Widens the CVD request so the stored bars reach a clicked candle. */
export function cvdRangeForAnchor(
  selected: CvdFetchRange,
  anchorSec: number | null,
  nowSec: number,
): CvdFetchRange {
  if (anchorSec == null || !Number.isFinite(anchorSec) || !Number.isFinite(nowSec)) return selected;
  const age = Math.max(0, nowSec - anchorSec);
  let need: CvdFetchRange = '90d';
  for (const id of CVD_FETCH_RANGES) {
    if (age <= CVD_RANGE_SEC[id]) {
      need = id;
      break;
    }
  }
  return CVD_FETCH_RANGES.indexOf(need) > CVD_FETCH_RANGES.indexOf(selected) ? need : selected;
}

/** Bar size of a CVD range. Unknown labels stay on the 24h 5-minute bar. */
export function cvdBarSeconds(bar: string): number {
  switch (bar) {
    case '30m':
      return 30 * 60;
    case '2h':
      return 2 * 60 * 60;
    case '4h':
      return 4 * 60 * 60;
    default:
      return 5 * 60;
  }
}

/** Candle open that contains `timeSec`. A click between candles does not anchor. */
export function snapCandleOpen(
  candles: { time: number }[],
  timeSec: number,
  barDurationSec: number,
): number | null {
  if (!Number.isFinite(timeSec) || barDurationSec <= 0) return null;
  for (let i = candles.length - 1; i >= 0; i -= 1) {
    const open = candles[i]?.time;
    if (open == null || !Number.isFinite(open)) continue;
    if (timeSec >= open && timeSec < open + barDurationSec) return open;
  }
  return null;
}

export function formatUtcMinute(sec: number): string {
  const d = new Date(sec * 1000);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

/**
 * True when no stored bar starts inside one CVD bar of the anchor.
 * The line then begins later than the candle the reader clicked.
 */
export function cvdAnchorStartsLate(
  points: { time: number }[],
  anchorSec: number,
  barSec: number,
): boolean {
  if (!Number.isFinite(anchorSec) || barSec <= 0) return true;
  let first: number | null = null;
  for (const point of points) {
    if (!Number.isFinite(point.time) || point.time < anchorSec) continue;
    if (first == null || point.time < first) first = point.time;
  }
  if (first == null) return true;
  return first > anchorSec + barSec;
}

/**
 * Strongest estimated long level at or below last, and short level at or above
 * last, for each of 10× / 25× / 50× / 100×. Same price on one side collapses
 * to one line.
 */
export function liquidationChartLevels(
  report: MarketLiquidationLevels | undefined,
): ChartLiqLevel[] {
  const levels = report?.levels ?? [];
  if (levels.length === 0 || report?.kind === 'totals') return [];
  const last = parseTapeNumber(report?.lastPrice);
  const grouped = new Map<string, ChartLiqLevel>();

  const take = (side: 'long' | 'short', leverage: number, price: number) => {
    const key = `${side}:${price}`;
    const prev = grouped.get(key);
    if (prev) {
      if (!prev.leverages.includes(leverage)) prev.leverages.push(leverage);
      return;
    }
    grouped.set(key, {
      id: `liq-${side}-${price}`,
      price,
      side,
      leverages: [leverage],
    });
  };

  for (const lev of LIQ_LEVERAGES) {
    let bestLong: { price: number; n: number } | null = null;
    let bestShort: { price: number; n: number } | null = null;
    for (const bar of levels) {
      const price = parseTapeNumber(bar.price);
      if (price == null) continue;
      const slice = bar.byLeverage?.find((s) => s.leverage === lev);
      const longN = parseTapeNumber(slice?.longNotional) ?? 0;
      const shortN = parseTapeNumber(slice?.shortNotional) ?? 0;
      if (longN > 0 && (last == null || price <= last) && (!bestLong || longN > bestLong.n)) {
        bestLong = { price, n: longN };
      }
      if (shortN > 0 && (last == null || price >= last) && (!bestShort || shortN > bestShort.n)) {
        bestShort = { price, n: shortN };
      }
    }
    if (bestLong) take('long', lev, bestLong.price);
    if (bestShort) take('short', lev, bestShort.price);
  }

  if (grouped.size === 0) {
    let bestLong: { price: number; n: number } | null = null;
    let bestShort: { price: number; n: number } | null = null;
    for (const bar of levels) {
      const price = parseTapeNumber(bar.price);
      if (price == null) continue;
      const longN = parseTapeNumber(bar.longNotional) ?? 0;
      const shortN = parseTapeNumber(bar.shortNotional) ?? 0;
      if (longN > 0 && (last == null || price <= last) && (!bestLong || longN > bestLong.n)) {
        bestLong = { price, n: longN };
      }
      if (shortN > 0 && (last == null || price >= last) && (!bestShort || shortN > bestShort.n)) {
        bestShort = { price, n: shortN };
      }
    }
    if (bestLong) take('long', 0, bestLong.price);
    if (bestShort) take('short', 0, bestShort.price);
  }

  return [...grouped.values()].map((line) => ({
    ...line,
    leverages: line.leverages.filter((n) => n > 0).sort((a, b) => b - a),
  }));
}
