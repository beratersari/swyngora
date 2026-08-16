import {
  COB_MAX_WIDTH,
  COB_MIN_WIDTH,
  DEPTH_ASK,
  DEPTH_BED,
  DEPTH_BID,
  HEATMAP_PAD,
  MAX_PRICE_BUCKETS,
} from './OrderHeatmap.constants';
import type {
  ColumnRect,
  HeatHover,
  HeatLayout,
  OrderHeatmapColumn,
  OrderHeatmapData,
  OrderHeatmapLevel,
} from './OrderHeatmap.types';

export function parseBookNumber(raw: string | undefined): number {
  const n = Number.parseFloat(raw ?? '');
  return Number.isFinite(n) ? n : NaN;
}

export function intensityFromNotional(notional: number, max: number): number {
  if (!(max > 0) || !(notional > 0)) return 0;
  const t = Math.log10(1 + notional) / Math.log10(1 + max);
  return Math.min(1, Math.max(0, t));
}

type Rgb = { r: number; g: number; b: number };

function mixRgb(a: Rgb, b: Rgb, t: number): Rgb {
  const u = Math.min(1, Math.max(0, t));
  return {
    r: Math.round(a.r + (b.r - a.r) * u),
    g: Math.round(a.g + (b.g - a.g) * u),
    b: Math.round(a.b + (b.b - a.b) * u),
  };
}

function smoothstep(t: number): number {
  const x = Math.min(1, Math.max(0, t));
  return x * x * (3 - 2 * x);
}

function rgbString(c: Rgb): string {
  return `rgb(${c.r}, ${c.g}, ${c.b})`;
}

/** Brand depth color: empty bed → green bids / red asks. */
export function depthColor(side: 'bid' | 'ask' | null, intensity: number): string {
  if (!side || intensity <= 0) return rgbString(DEPTH_BED);
  const tint = side === 'bid' ? DEPTH_BID : DEPTH_ASK;
  const from = mixRgb(DEPTH_BED, tint, 0.1);
  return rgbString(mixRgb(from, tint, smoothstep(intensity)));
}

/** Soften hard stripes so adjacent price rows blend. */
export function bloomIntensities(values: number[]): number[] {
  if (values.length === 0) return [];
  const out = values.slice();
  for (let i = 0; i < values.length; i += 1) {
    const prev = values[i - 1] ?? 0;
    const next = values[i + 1] ?? 0;
    out[i] = values[i] * 0.62 + prev * 0.19 + next * 0.19;
  }
  return out;
}

export function gradientStops(
  buckets: { side: 'bid' | 'ask' | null }[],
  heats: number[],
  layout: HeatLayout,
  step: number,
): { t: number; color: string }[] {
  const raw: { t: number; color: string }[] = [];
  for (let i = 0; i < buckets.length; i += 1) {
    const y = priceToY(buckets[i].price + step / 2, layout);
    const t = (y - layout.plotY) / Math.max(1, layout.plotH);
    raw.push({ t: Math.min(1, Math.max(0, t)), color: depthColor(buckets[i].side, heats[i] ?? 0) });
  }
  raw.sort((a, b) => a.t - b.t);
  const out: { t: number; color: string }[] = [];
  for (const stop of raw) {
    const prev = out[out.length - 1];
    if (prev && Math.abs(prev.t - stop.t) < 1e-4) {
      prev.color = stop.color;
      continue;
    }
    out.push(stop);
  }
  if (out.length === 0) return [{ t: 0, color: rgbString(DEPTH_BED) }, { t: 1, color: rgbString(DEPTH_BED) }];
  if (out[0].t > 0) out.unshift({ t: 0, color: out[0].color });
  if (out[out.length - 1].t < 1) out.push({ t: 1, color: out[out.length - 1].color });
  return out;
}

export function parseGroupSize(raw: string | undefined): number {
  const n = parseBookNumber(raw);
  return n > 0 ? n : 0;
}

type ParsedLevel = { price: number; notional: number; isWall: boolean; side: 'bid' | 'ask' };

function parseLevels(levels: OrderHeatmapLevel[] | undefined, side: 'bid' | 'ask'): ParsedLevel[] {
  const out: ParsedLevel[] = [];
  for (const lv of levels ?? []) {
    const price = parseBookNumber(lv.price);
    const notional = parseBookNumber(lv.notional);
    if (!(price > 0) || !(notional > 0)) continue;
    out.push({ price, notional, isWall: Boolean(lv.isWall), side });
  }
  return out;
}

export function columnTimes(columns: OrderHeatmapColumn[] | undefined): number[] {
  const out: number[] = [];
  for (const col of columns ?? []) {
    const timeMs = Date.parse(col.t ?? '');
    out.push(Number.isFinite(timeMs) ? timeMs : NaN);
  }
  return out;
}

export function priceExtent(columns: OrderHeatmapColumn[] | undefined): { min: number; max: number } | null {
  let min = Number.POSITIVE_INFINITY;
  let max = Number.NEGATIVE_INFINITY;
  for (const col of columns ?? []) {
    const mid = parseBookNumber(col.mid);
    if (mid > 0) {
      if (mid < min) min = mid;
      if (mid > max) max = mid;
    }
    for (const lv of [...(col.bids ?? []), ...(col.asks ?? [])]) {
      const price = parseBookNumber(lv.price);
      if (!(price > 0)) continue;
      if (price < min) min = price;
      if (price > max) max = price;
    }
  }
  if (!Number.isFinite(min) || !Number.isFinite(max)) return null;
  if (min === max) {
    const pad = Math.max(Math.abs(min) * 0.001, 1e-8);
    return { min: min - pad, max: max + pad };
  }
  const pad = (max - min) * 0.03;
  return { min: min - pad, max: max + pad };
}

export function buildPriceGrid(min: number, max: number, groupSize: number): { prices: number[]; step: number } {
  const rawStep = groupSize > 0 ? groupSize : (max - min) / 80;
  let step = rawStep > 0 ? rawStep : (max - min) / 80;
  if ((max - min) / step > MAX_PRICE_BUCKETS) {
    step = (max - min) / MAX_PRICE_BUCKETS;
  }
  const start = Math.floor(min / step) * step;
  const prices: number[] = [];
  const last = max + step * 0.25;
  for (let p = start; p <= last; p += step) {
    prices.push(p);
    if (prices.length >= MAX_PRICE_BUCKETS + 4) break;
  }
  if (prices.length === 0) prices.push(min, max);
  return { prices, step };
}

export function peakNotional(columns: OrderHeatmapColumn[] | undefined): number {
  let max = 0;
  for (const col of columns ?? []) {
    for (const lv of [...(col.bids ?? []), ...(col.asks ?? [])]) {
      const n = parseBookNumber(lv.notional);
      if (n > max) max = n;
    }
  }
  return max;
}

export function columnRects(count: number, plotX: number, plotW: number, times: number[]): ColumnRect[] {
  if (count <= 0 || plotW <= 0) return [];
  if (count === 1) {
    return [{ index: 0, x: plotX, w: plotW, timeMs: times[0] ?? NaN, isCob: true }];
  }
  const cob = Math.min(COB_MAX_WIDTH, Math.max(COB_MIN_WIDTH, plotW * 0.2));
  const gap = 3;
  const histW = Math.max(1, plotW - cob - gap);
  const colW = histW / (count - 1);
  const out: ColumnRect[] = [];
  for (let i = 0; i < count - 1; i += 1) {
    out.push({
      index: i,
      x: plotX + i * colW,
      w: colW,
      timeMs: times[i] ?? NaN,
      isCob: false,
    });
  }
  out.push({
    index: count - 1,
    x: plotX + histW + gap,
    w: cob,
    timeMs: times[count - 1] ?? NaN,
    isCob: true,
  });
  return out;
}

export function buildLayout(
  data: OrderHeatmapData | undefined,
  width: number,
  height: number,
): HeatLayout | null {
  const columns = data?.columns ?? [];
  const extent = priceExtent(columns);
  if (!extent) return null;
  const group = parseGroupSize(data?.groupSize);
  const { prices, step } = buildPriceGrid(extent.min, extent.max, group);
  const plotX = HEATMAP_PAD.left;
  const plotY = HEATMAP_PAD.top;
  const scaleW = 10;
  const scaleX = Math.max(plotX + 40, width - HEATMAP_PAD.right);
  const plotW = Math.max(1, scaleX - 8 - plotX);
  const plotH = Math.max(1, height - HEATMAP_PAD.top - HEATMAP_PAD.bottom);
  const times = columnTimes(columns);
  return {
    plotX,
    plotY,
    plotW,
    plotH,
    scaleX,
    scaleW,
    minPrice: prices[0] ?? extent.min,
    maxPrice: prices[prices.length - 1] ?? extent.max,
    step,
    prices,
    rects: columnRects(columns.length, plotX, plotW, times),
    peak: peakNotional(columns),
  };
}

export function priceToY(price: number, layout: HeatLayout): number {
  const span = Math.max(1e-12, layout.maxPrice - layout.minPrice);
  const t = (price - layout.minPrice) / span;
  return layout.plotY + (1 - t) * layout.plotH;
}

export type Bucket = {
  price: number;
  notional: number;
  side: 'bid' | 'ask' | null;
  isWall: boolean;
};

function nearestStep(price: number, step: number): number {
  if (!(step > 0)) return price;
  return Math.round(price / step) * step;
}

export function bucketsForColumn(col: OrderHeatmapColumn, prices: number[], step: number): Bucket[] {
  const bids = parseLevels(col.bids, 'bid');
  const asks = parseLevels(col.asks, 'ask');
  const byPrice = new Map<number, Bucket>();
  const take = (lv: ParsedLevel) => {
    const key = nearestStep(lv.price, step);
    const prev = byPrice.get(key);
    if (!prev || lv.notional > prev.notional) {
      byPrice.set(key, { price: key, notional: lv.notional, side: lv.side, isWall: lv.isWall });
    } else if (prev && lv.isWall) {
      prev.isWall = true;
    }
  };
  for (const lv of bids) take(lv);
  for (const lv of asks) take(lv);
  return prices.map((price) => byPrice.get(nearestStep(price, step)) ?? { price, notional: 0, side: null, isWall: false });
}

export function formatHeatPrice(price: number, groupSize: number): string {
  if (!Number.isFinite(price)) return '—';
  if (groupSize >= 1) return price.toFixed(0);
  if (groupSize >= 0.1) return price.toFixed(1);
  if (groupSize >= 0.01) return price.toFixed(2);
  if (groupSize > 0) {
    const digits = Math.min(8, Math.max(2, Math.ceil(-Math.log10(groupSize))));
    return price.toFixed(digits);
  }
  if (Math.abs(price) >= 1000) return price.toFixed(0);
  if (Math.abs(price) >= 1) return price.toFixed(2);
  return price.toPrecision(4);
}

export function formatHeatNotional(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '—';
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  if (n >= 10) return n.toFixed(0);
  return n.toFixed(2);
}

export function formatCollectedSpan(from?: string, to?: string): string | null {
  const start = Date.parse(from ?? '');
  const end = Date.parse(to ?? '');
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return null;
  const sec = Math.max(1, Math.round((end - start) / 1000));
  if (sec < 60) return `${sec}s`;
  const minutes = Math.floor(sec / 60);
  const rest = sec % 60;
  return rest === 0 ? `${minutes}m` : `${minutes}m ${rest}s`;
}

export function formatHeatTime(timeMs: number): string {
  if (!Number.isFinite(timeMs)) return '—';
  const d = new Date(timeMs);
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  const ss = String(d.getSeconds()).padStart(2, '0');
  return `${hh}:${mm}:${ss}`;
}

export function hitTest(
  px: number,
  py: number,
  data: OrderHeatmapData | undefined,
  layout: HeatLayout,
): HeatHover | null {
  if (py < layout.plotY || py > layout.plotY + layout.plotH) return null;
  const rect = [...layout.rects].reverse().find((r) => px >= r.x && px <= r.x + r.w);
  if (!rect) return null;
  const col = data?.columns?.[rect.index];
  if (!col) return null;
  const span = Math.max(1e-12, layout.maxPrice - layout.minPrice);
  const t = 1 - (py - layout.plotY) / layout.plotH;
  const price = layout.minPrice + t * span;
  const buckets = bucketsForColumn(col, layout.prices, layout.step);
  let best = buckets[0];
  let bestDist = Number.POSITIVE_INFINITY;
  for (const b of buckets) {
    const d = Math.abs(b.price - price);
    if (d < bestDist) {
      bestDist = d;
      best = b;
    }
  }
  return {
    x: px,
    y: py,
    timeMs: rect.timeMs,
    price: best?.price ?? price,
    mid: parseBookNumber(col.mid) || null,
    bid: best?.side === 'bid' ? best.notional : 0,
    ask: best?.side === 'ask' ? best.notional : 0,
    bidWall: best?.side === 'bid' && best.isWall,
    askWall: best?.side === 'ask' && best.isWall,
  };
}

export function yTicks(layout: HeatLayout, count = 6): number[] {
  const n = Math.max(2, count);
  const out: number[] = [];
  for (let i = 0; i < n; i += 1) {
    const t = i / (n - 1);
    out.push(layout.minPrice + t * (layout.maxPrice - layout.minPrice));
  }
  return out;
}

export function xTickRects(layout: HeatLayout): ColumnRect[] {
  if (layout.rects.length <= 4) return layout.rects;
  const first = layout.rects[0];
  const last = layout.rects[layout.rects.length - 1];
  const mid = layout.rects[Math.floor(layout.rects.length / 2)];
  return [first, mid, last];
}
