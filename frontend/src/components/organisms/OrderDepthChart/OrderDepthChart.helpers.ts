import type { OrderBookLevel, SpotOrderBook } from '@/libs/api';
import { DEPTH_PAD } from './OrderDepthChart.constants';
import type { DepthHover, DepthLayout, DepthMetric, DepthPoint, DepthSeries } from './OrderDepthChart.types';

export function parseDepthNumber(raw: string | number | null | undefined): number {
  if (raw == null || raw === '') return NaN;
  const n = typeof raw === 'number' ? raw : Number.parseFloat(raw);
  return Number.isFinite(n) ? n : NaN;
}

function levelDepth(lv: OrderBookLevel, metric: DepthMetric): number {
  const raw = metric === 'notional' ? lv.cumulativeNotional : lv.cumulative;
  const n = parseDepthNumber(raw);
  if (n > 0) return n;
  const fallback = metric === 'notional' ? lv.notional : lv.quantity;
  const f = parseDepthNumber(fallback);
  return f > 0 ? f : 0;
}

/** Build bid/ask polylines: mid is 0 depth; size grows as price moves away. */
export function buildDepthSeries(book: SpotOrderBook | undefined, metric: DepthMetric): DepthSeries | null {
  if (!book) return null;
  const mid =
    parseDepthNumber(book.lastPrice) ||
    (parseDepthNumber(book.bestBid) + parseDepthNumber(book.bestAsk)) / 2;
  if (!(mid > 0)) return null;

  const bids: DepthPoint[] = [];
  for (const lv of book.bids ?? []) {
    const price = parseDepthNumber(lv.price);
    const depth = levelDepth(lv, metric);
    if (!(price > 0) || !(depth > 0)) continue;
    bids.push({ price, depth });
  }
  const asks: DepthPoint[] = [];
  for (const lv of book.asks ?? []) {
    const price = parseDepthNumber(lv.price);
    const depth = levelDepth(lv, metric);
    if (!(price > 0) || !(depth > 0)) continue;
    asks.push({ price, depth });
  }
  if (bids.length === 0 && asks.length === 0) return null;

  bids.sort((a, b) => a.price - b.price);
  asks.sort((a, b) => a.price - b.price);

  let maxDepth = 0;
  for (const p of [...bids, ...asks]) {
    if (p.depth > maxDepth) maxDepth = p.depth;
  }
  if (!(maxDepth > 0)) return null;

  const minPrice = bids.length ? bids[0].price : mid;
  const maxPrice = asks.length ? asks[asks.length - 1].price : mid;
  return { mid, bids, asks, maxDepth, minPrice, maxPrice };
}

export function buildDepthLayout(series: DepthSeries, width: number, height: number): DepthLayout {
  return {
    plotX: DEPTH_PAD.left,
    plotY: DEPTH_PAD.top,
    plotW: Math.max(1, width - DEPTH_PAD.left - DEPTH_PAD.right),
    plotH: Math.max(1, height - DEPTH_PAD.top - DEPTH_PAD.bottom),
    minPrice: series.minPrice,
    maxPrice: series.maxPrice,
    maxDepth: series.maxDepth,
  };
}

export function priceToX(price: number, layout: DepthLayout): number {
  const span = Math.max(1e-12, layout.maxPrice - layout.minPrice);
  return layout.plotX + ((price - layout.minPrice) / span) * layout.plotW;
}

export function depthToY(depth: number, layout: DepthLayout): number {
  const t = Math.min(1, Math.max(0, depth / Math.max(layout.maxDepth, 1e-12)));
  return layout.plotY + layout.plotH * (1 - t);
}

export function xToPrice(x: number, layout: DepthLayout): number {
  const span = Math.max(1e-12, layout.maxPrice - layout.minPrice);
  const t = (x - layout.plotX) / Math.max(layout.plotW, 1e-12);
  return layout.minPrice + t * span;
}

function nearestPoint(price: number, points: DepthPoint[]): DepthPoint | null {
  if (!points.length) return null;
  let best = points[0];
  let bestD = Math.abs(points[0].price - price);
  for (let i = 1; i < points.length; i += 1) {
    const d = Math.abs(points[i].price - price);
    if (d < bestD) {
      best = points[i];
      bestD = d;
    }
  }
  return best;
}

export function hitTestDepth(
  x: number,
  y: number,
  series: DepthSeries,
  layout: DepthLayout,
): DepthHover | null {
  if (x < layout.plotX || x > layout.plotX + layout.plotW) return null;
  if (y < layout.plotY || y > layout.plotY + layout.plotH) return null;
  const price = xToPrice(x, layout);
  const side: 'bid' | 'ask' = price <= series.mid ? 'bid' : 'ask';
  const pt = nearestPoint(price, side === 'bid' ? series.bids : series.asks);
  if (!pt) return null;
  return {
    x,
    y,
    side,
    price: pt.price,
    depth: pt.depth,
  };
}

export function formatDepthAmount(value: number): string {
  if (!(value > 0) || !Number.isFinite(value)) return '—';
  const abs = Math.abs(value);
  if (abs >= 1e9) return `${(value / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${(value / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${(value / 1e3).toFixed(2)}K`;
  if (abs >= 1) return value.toLocaleString(undefined, { maximumFractionDigits: 4 });
  return value.toLocaleString(undefined, { maximumFractionDigits: 8 });
}

export function formatDepthPrice(value: number): string {
  if (!(value > 0) || !Number.isFinite(value)) return '—';
  if (value >= 1000) return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
  if (value >= 1) return value.toLocaleString(undefined, { maximumFractionDigits: 4 });
  return value.toLocaleString(undefined, { maximumFractionDigits: 8 });
}

export function yTicks(maxDepth: number): number[] {
  if (!(maxDepth > 0)) return [];
  const raw = maxDepth / 4;
  const mag = 10 ** Math.floor(Math.log10(raw));
  const step = Math.max(mag, Math.round(raw / mag) * mag);
  const out: number[] = [];
  for (let v = step; v < maxDepth * 0.98; v += step) out.push(v);
  return out.slice(0, 5);
}

export function xTicks(minPrice: number, maxPrice: number): number[] {
  const span = maxPrice - minPrice;
  if (!(span > 0)) return [minPrice];
  const step = span / 4;
  return [minPrice, minPrice + step, minPrice + 2 * step, minPrice + 3 * step, maxPrice];
}
