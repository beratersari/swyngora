import type {
  CandleChartFootprintLevel,
  CandleChartHeatColumn,
  CandleChartLiqBar,
  CandleChartLowerSeries,
} from '@/components/molecules/CandleChartHost';
import type { ChartLinePoint } from '@/libs/utils';
import type {
  FlowAtPricesReport,
  HistorySpan,
  MarketLiquidationLevels,
  MarketLongShortRatio,
  OrderBookHeatmap,
} from '@/libs/api';
import { parseTapeNumber, unixSeconds } from './chartTape';

const HEAT_COLUMNS = 90;
const HEAT_LEVELS = 24;
const FOOTPRINT_ROWS = 48;
const LIQ_ROWS = 12;

/** Map a visible candle span onto a flow-at-prices window. */
export function footprintWindow(spanSec: number): '5m' | '15m' | '1h' | '4h' | '1d' | '3d' | '7d' {
  if (!Number.isFinite(spanSec) || spanSec <= 5 * 60) return '5m';
  if (spanSec <= 15 * 60) return '15m';
  if (spanSec <= 60 * 60) return '1h';
  if (spanSec <= 4 * 60 * 60) return '4h';
  if (spanSec <= 24 * 60 * 60) return '1d';
  if (spanSec <= 3 * 24 * 60 * 60) return '3d';
  return '7d';
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

export function crowdTopSeries(
  report: MarketLongShortRatio | undefined,
  labels: { crowd: string; top: string },
  colors: { crowd: string; top: string },
  pane: number,
): CandleChartLowerSeries[] {
  const lines = crowdTopLines(report);
  if (lines.crowd.length === 0 && lines.top.length === 0) return [];
  const out: CandleChartLowerSeries[] = [];
  if (lines.crowd.length > 0) {
    out.push({
      id: 'crowd-long',
      pane,
      kind: 'line',
      title: labels.crowd,
      color: colors.crowd,
      data: lines.crowd,
    });
  }
  if (lines.top.length > 0) {
    out.push({
      id: 'top-long',
      pane,
      kind: 'line',
      title: labels.top,
      color: colors.top,
      data: lines.top,
    });
  }
  return out;
}

export type RecordedFutures = { exchange: string; spanSec: number };

/** Futures open-interest rows that have a stored span. */
export function futuresRecorded(span: HistorySpan | undefined): RecordedFutures[] {
  return (span?.series ?? [])
    .filter((row) => row.kind === 'futures_oi' && (row.spanSec ?? 0) > 0 && row.exchange)
    .map((row) => ({ exchange: String(row.exchange), spanSec: row.spanSec ?? 0 }));
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
