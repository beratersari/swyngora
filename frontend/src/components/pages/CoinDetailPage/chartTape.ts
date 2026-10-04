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

/** Futures CVD series: combined when it has points, otherwise the first venue that does. */
export function futuresCvdSeries(report: MarketCvd | undefined): {
  points: TapeDeltaPoint[];
  dayFrom: number | null;
  blockFrom: number | null;
  complete: boolean;
  range: string;
  bar: string;
} {
  const venue =
    report?.combined?.points && report.combined.points.length > 0
      ? report.combined
      : report?.venues?.find((v) => (v.points?.length ?? 0) > 0);
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
    range: report?.range ?? '24h',
    bar: report?.bar ?? '5m',
  };
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
