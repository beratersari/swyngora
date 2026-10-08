import type { MarketLiquidationLevels } from '@/libs/api';
import type { LevelRow, LeverageSlice, TimeRow } from './LiquidationBarChart.types';
import { LIQ_LEVERAGE_ORDER } from './LiquidationBarChart.constants';

function sliceConf(raw: number | undefined): number {
  if (raw == null || !Number.isFinite(raw) || raw <= 0) return 1;
  if (raw > 1) return 1;
  return raw;
}

export function bandHitLabel(
  buckets: NonNullable<LevelsReviewVenue['byLeverage']> | undefined,
  lev: number,
): { long?: string; short?: string } {
  const out: { long?: string; short?: string } = {};
  for (const row of buckets ?? []) {
    if (row.leverage !== lev || !row.validated || row.hitRate == null || !Number.isFinite(row.hitRate)) continue;
    const label = formatHitRate(row.hitRate);
    if (row.side === 'long') out.long = label;
    if (row.side === 'short') out.short = label;
  }
  return out;
}

export function legendOpacity(
  buckets: NonNullable<LevelsReviewVenue['byLeverage']> | undefined,
  lev: 10 | 25 | 50 | 100,
): number {
  const rows = (buckets ?? []).filter((b) => b.leverage === lev);
  const weights = rows.map((b) => b.confidence).filter((c): c is number => c != null && c > 0);
  if (!weights.length) return 1;
  const avg = weights.reduce((a, b) => a + b, 0) / weights.length;
  return 0.28 + 0.72 * avg;
}

export function sliceConfidence(row: LevelRow, leverage: 10 | 25 | 50 | 100, side: 'long' | 'short'): number {
  const sl = row.cumByLeverage.find((s) => s.leverage === leverage) ?? row.byLeverage.find((s) => s.leverage === leverage);
  if (!sl) return 1;
  return side === 'long' ? sl.longConf : sl.shortConf;
}

export function parseNotional(value: string | undefined): number {
  if (!value) return 0;
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

function parseSlices(
  raw: NonNullable<NonNullable<MarketLiquidationLevels['levels']>[number]['byLeverage']> | undefined,
): LeverageSlice[] {
  return (raw ?? [])
    .map((sl) => ({
      leverage: sl.leverage as 10 | 25 | 50 | 100,
      longN: parseNotional(sl.longNotional),
      shortN: parseNotional(sl.shortNotional),
      longConf: sliceConf(sl.longConfidence),
      shortConf: sliceConf(sl.shortConfidence),
    }))
    .filter((sl) => sl.leverage === 10 || sl.leverage === 25 || sl.leverage === 50 || sl.leverage === 100);
}

export function toLevelRows(data?: MarketLiquidationLevels | null): LevelRow[] {
  const rows = (data?.levels ?? [])
    .map((lv) => ({
      price: parseNotional(lv.price),
      longN: parseNotional(lv.longNotional),
      shortN: parseNotional(lv.shortNotional),
      totalN: parseNotional(lv.totalNotional),
      cumLong: parseNotional(lv.cumLong),
      cumShort: parseNotional(lv.cumShort),
      cumTotal: parseNotional(lv.cumTotal),
      byLeverage: parseSlices(lv.byLeverage),
      cumByLeverage: parseSlices(lv.cumByLeverage),
    }))
    .filter((row) => Number.isFinite(row.price) && row.price > 0);
  return fillCumProfile(rows, parseNotional(data?.lastPrice));
}

/** Walk from last so the silhouette is the CoinGlass cumulative, even if the API omitted cumByLeverage. */
export function fillCumProfile(rows: LevelRow[], lastPrice: number): LevelRow[] {
  if (!rows.length) return rows;
  const last = lastPrice > 0 ? lastPrice : rows[Math.floor(rows.length / 2)]?.price ?? 0;
  const accAbove: Record<10 | 25 | 50 | 100, { longN: number; shortN: number }> = {
    10: { longN: 0, shortN: 0 },
    25: { longN: 0, shortN: 0 },
    50: { longN: 0, shortN: 0 },
    100: { longN: 0, shortN: 0 },
  };
  const accBelow = {
    10: { longN: 0, shortN: 0 },
    25: { longN: 0, shortN: 0 },
    50: { longN: 0, shortN: 0 },
    100: { longN: 0, shortN: 0 },
  };
  const above: LevelRow['cumByLeverage'][] = rows.map(() => []);
  const below: LevelRow['cumByLeverage'][] = rows.map(() => []);
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    const row = rows[i];
    if (!row || (last > 0 && row.price <= last)) continue;
    for (const lev of LIQ_LEVERAGE_ORDER) {
      const sl = row.byLeverage.find((s) => s.leverage === lev);
      accAbove[lev].longN += sl?.longN ?? 0;
      accAbove[lev].shortN += sl?.shortN ?? 0;
    }
    above[i] = LIQ_LEVERAGE_ORDER.map((lev) => {
      const sl = row.byLeverage.find((s) => s.leverage === lev);
      return { leverage: lev, ...accAbove[lev], longConf: sl?.longConf ?? 1, shortConf: sl?.shortConf ?? 1 };
    });
  }
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i];
    if (!row || (last > 0 && row.price >= last)) continue;
    for (const lev of LIQ_LEVERAGE_ORDER) {
      const sl = row.byLeverage.find((s) => s.leverage === lev);
      accBelow[lev].longN += sl?.longN ?? 0;
      accBelow[lev].shortN += sl?.shortN ?? 0;
    }
    below[i] = LIQ_LEVERAGE_ORDER.map((lev) => {
      const sl = row.byLeverage.find((s) => s.leverage === lev);
      return { leverage: lev, ...accBelow[lev], longConf: sl?.longConf ?? 1, shortConf: sl?.shortConf ?? 1 };
    });
  }
  return rows.map((row, i) => {
    const walked = last <= 0 || row.price >= last ? above[i] : below[i];
    const cumByLeverage =
      row.cumByLeverage.length === LIQ_LEVERAGE_ORDER.length ? row.cumByLeverage : (walked ?? []);
    const cumLong = cumByLeverage.reduce((s, sl) => s + sl.longN, 0);
    const cumShort = cumByLeverage.reduce((s, sl) => s + sl.shortN, 0);
    return {
      ...row,
      cumByLeverage,
      cumLong: row.cumLong > 0 ? row.cumLong : cumLong,
      cumShort: row.cumShort > 0 ? row.cumShort : cumShort,
      cumTotal: row.cumTotal > 0 ? row.cumTotal : cumLong + cumShort,
    };
  });
}

export function sliceNotional(row: LevelRow, leverage: 10 | 25 | 50 | 100, side: 'long' | 'short'): number {
  const sl = row.byLeverage.find((s) => s.leverage === leverage);
  if (!sl) return 0;
  return side === 'long' ? sl.longN : sl.shortN;
}

export function cumSliceNotional(row: LevelRow, leverage: 10 | 25 | 50 | 100, side: 'long' | 'short'): number {
  const sl = row.cumByLeverage.find((s) => s.leverage === leverage);
  if (!sl) return 0;
  return side === 'long' ? sl.longN : sl.shortN;
}

export function maxCum(rows: LevelRow[]): number {
  let m = 0;
  for (const row of rows) {
    m = Math.max(m, row.cumTotal, row.totalN);
  }
  return m;
}

export function formatAxisNotional(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '0';
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(n >= 10_000_000_000 ? 0 : 1)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n >= 10_000_000 ? 0 : 1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(n >= 10_000 ? 0 : 1)}K`;
  return n.toFixed(0);
}

export function formatAxisPrice(n: number): string {
  if (!Number.isFinite(n)) return '—';
  if (n >= 1000) return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
  if (n >= 1) return n.toLocaleString(undefined, { maximumFractionDigits: 2 });
  return n.toLocaleString(undefined, { maximumFractionDigits: 6 });
}

export function barSide(row: LevelRow, lastPrice: number): 'long' | 'short' {
  if (lastPrice > 0 && row.price >= lastPrice) return 'short';
  if (lastPrice > 0) return 'long';
  return row.shortN >= row.longN ? 'short' : 'long';
}

/** First stored print when earlier bars in the requested range are empty. */
export function storedPrintStart(rows: TimeRow[]): number | null {
  const first = rows.findIndex((row) => row.totalN > 0);
  if (first <= 0) return null;
  const at = rows[first]?.t ?? 0;
  return at > 0 ? at : null;
}

export function formatStoredPrintStart(at: number): string {
  return new Date(at).toISOString().slice(0, 16).replace('T', ' ');
}

export function toTimeRows(data?: MarketLiquidationLevels | null): TimeRow[] {
  return (data?.bars ?? []).map((bar) => {
    const t = Date.parse(bar.t ?? '');
    const d = Number.isFinite(t) ? new Date(t) : null;
    const label = d
      ? d.toISOString().slice(11, 16)
      : '';
    return {
      t: Number.isFinite(t) ? t : 0,
      label,
      longN: parseNotional(bar.longNotional),
      shortN: parseNotional(bar.shortNotional),
      totalN: parseNotional(bar.totalNotional),
    };
  });
}

export function maxSide(rows: { longN: number; shortN: number }[]): number {
  let m = 0;
  for (const row of rows) {
    m = Math.max(m, row.longN, row.shortN);
  }
  return m;
}

export function maxTotal(rows: { totalN: number }[]): number {
  let m = 0;
  for (const row of rows) {
    m = Math.max(m, row.totalN);
  }
  return m;
}

export function isLevelsKind(data?: MarketLiquidationLevels | null): boolean {
  return data?.kind === 'levels';
}

export type LevelsReviewVenue = NonNullable<
  NonNullable<MarketLiquidationLevels['review']>['combined']
>;

export function pickLevelsReview(data?: MarketLiquidationLevels | null): LevelsReviewVenue | undefined {
  const review = data?.review;
  if (!review) return undefined;
  if (data?.exchange === 'binance') return review.binance;
  if (data?.exchange === 'bybit') return review.bybit;
  if (data?.exchange === 'okx') return review.okx;
  return review.combined ?? review.binance;
}

export function formatHitRate(n: number | undefined): string {
  if (n == null || !Number.isFinite(n)) return '—';
  return `${Math.round(n * 100)}%`;
}

export function newestLevelSamples(venue?: LevelsReviewVenue): NonNullable<LevelsReviewVenue['samples']> {
  const rows = [...(venue?.samples ?? [])];
  rows.sort((a, b) => Date.parse(b.time ?? '') - Date.parse(a.time ?? ''));
  return rows.slice(0, 8);
}

/** Live map, or a frozen past estimate that keeps shorts above that sample's last. */
export function frozenLevelsView(
  data: MarketLiquidationLevels | null | undefined,
  samples: NonNullable<LevelsReviewVenue['samples']>,
  asOf: string,
): MarketLiquidationLevels | null | undefined {
  if (!data || asOf === 'live') return data;
  const sample = samples.find((s) => s.time === asOf);
  if (!sample?.levels?.length) return data;
  return {
    ...data,
    lastPrice: sample.lastPrice,
    levels: sample.levels,
  };
}

export function bandStatusLabel(status: string | undefined): string {
  if (status === 'hit') return 'hit';
  if (status === 'miss') return 'miss';
  if (status === 'pending') return 'pending';
  return status || '—';
}
