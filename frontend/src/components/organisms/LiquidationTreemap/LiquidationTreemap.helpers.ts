import type { LiquidationTreemapCoin, LiquidationTreemapTile } from './LiquidationTreemap.types';
import {
  LIQ_TREEMAP_BANDS,
  LIQ_TREEMAP_GUTTER,
  LIQ_TREEMAP_INK,
  LIQ_TREEMAP_INK_ON_NEUTRAL,
  LIQ_TREEMAP_MAX_TILES,
  LIQ_TREEMAP_NEUTRAL,
} from './LiquidationTreemap.constants';

export function parseNotional(value: string | number | null | undefined): number {
  if (value == null || value === '') return 0;
  const n = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function coinLongShare(coin: LiquidationTreemapCoin): number {
  const longN = parseNotional(coin.longNotional);
  const shortN = parseNotional(coin.shortNotional);
  const total = longN + shortN;
  if (total <= 0) return 0.5;
  return longN / total;
}

export function coinBase(symbol: string, base?: string | null): string {
  const fromApi = (base ?? '').trim();
  if (fromApi) return fromApi;
  return symbol.replace(/USDT$|USDC$/i, '') || symbol;
}

export function tileDensity(w: number, h: number): 'full' | 'compact' | 'ticker' | 'micro' {
  if (w >= 72 && h >= 52) return 'full';
  if (w >= 48 && h >= 32) return 'compact';
  if (w >= 28 && h >= 20) return 'ticker';
  return 'micro';
}

export function dominanceFill(longShare: number): string {
  if (!Number.isFinite(longShare)) return LIQ_TREEMAP_NEUTRAL;
  for (const band of LIQ_TREEMAP_BANDS) {
    if (longShare <= band.upTo) return band.fill;
  }
  return LIQ_TREEMAP_BANDS[LIQ_TREEMAP_BANDS.length - 1]!.fill;
}

export function tileInk(fill: string): string {
  return fill === LIQ_TREEMAP_NEUTRAL ? LIQ_TREEMAP_INK_ON_NEUTRAL : LIQ_TREEMAP_INK;
}

export function hoverCardOrigin(
  mouseX: number,
  mouseY: number,
  frameW: number,
  frameH: number,
  cardW = 210,
  cardH = 140,
): { x: number; y: number } {
  const pad = 10;
  let x = mouseX + 16;
  let y = mouseY + 16;
  if (x + cardW > frameW - pad) x = mouseX - cardW - 12;
  if (y + cardH > frameH - pad) y = mouseY - cardH - 12;
  return { x: Math.max(pad, x), y: Math.max(pad, y) };
}

type Rect = { x: number; y: number; w: number; h: number };
type Seed = Omit<LiquidationTreemapTile, 'x' | 'y' | 'w' | 'h'>;

export function toTreemapTiles(
  coins: LiquidationTreemapCoin[],
  width: number,
  height: number,
): LiquidationTreemapTile[] {
  if (width <= 0 || height <= 0) return [];
  const ranked = coins
    .filter((c) => c.symbol)
    .map((c) => ({
      ...c,
      symbol: c.symbol!,
      weight: Math.max(0, parseNotional(c.totalNotional)),
      longShare: coinLongShare(c),
    }))
    .filter((c) => c.weight > 0)
    .sort((a, b) => b.weight - a.weight)
    .slice(0, LIQ_TREEMAP_MAX_TILES);
  const peak = ranked[0]?.weight ?? 0;
  const floor = peak > 0 ? peak * 0.002 : 1;
  const prepared: Seed[] = ranked.map((it) => ({ ...it, weight: Math.max(it.weight, floor) }));
  if (!prepared.length) return [];
  return squarify(prepared, { x: 0, y: 0, w: width, h: height });
}

function squarify(items: Seed[], rect: Rect): LiquidationTreemapTile[] {
  const nodes = items.slice();
  let remaining = nodes.reduce((s, it) => s + it.weight, 0);
  if (remaining <= 0) return [];
  let { x, y, w, h } = rect;
  const out: LiquidationTreemapTile[] = [];
  let i0 = 0;
  const n = nodes.length;
  while (i0 < n && remaining > 0 && w > 1 && h > 1) {
    let i1 = i0 + 1;
    let sum = nodes[i0]!.weight;
    let minV = sum;
    let maxV = sum;
    const short = Math.min(w, h);
    const long = Math.max(w, h);
    const alpha = long / (short * remaining);
    let beta = sum * sum * alpha;
    let best = Math.max(maxV / beta, beta / minV);
    for (; i1 < n; i1++) {
      const v = nodes[i1]!.weight;
      const nextSum = sum + v;
      const nextMin = Math.min(minV, v);
      const nextMax = Math.max(maxV, v);
      const nextBeta = nextSum * nextSum * alpha;
      const ratio = Math.max(nextMax / nextBeta, nextBeta / nextMin);
      if (ratio > best) break;
      sum = nextSum;
      minV = nextMin;
      maxV = nextMax;
      best = ratio;
    }
    const row = nodes.slice(i0, i1);
    if (h < w) {
      const rw = w * (sum / remaining);
      let cy = y;
      for (const it of row) {
        const rh = h * (it.weight / sum);
        out.push({ ...it, x, y: cy, w: rw, h: rh });
        cy += rh;
      }
      x += rw;
      w -= rw;
    } else {
      const rh = h * (sum / remaining);
      let cx = x;
      for (const it of row) {
        const rw = w * (it.weight / sum);
        out.push({ ...it, x: cx, y, w: rw, h: rh });
        cx += rw;
      }
      y += rh;
      h -= rh;
    }
    remaining -= sum;
    i0 = i1;
  }
  return out.map(insetTile);
}

function insetTile(tile: LiquidationTreemapTile): LiquidationTreemapTile {
  const g = LIQ_TREEMAP_GUTTER;
  const x = tile.x + g / 2;
  const y = tile.y + g / 2;
  const w = Math.max(1, tile.w - g);
  const h = Math.max(1, tile.h - g);
  const left = Math.round(x);
  const top = Math.round(y);
  return {
    ...tile,
    x: left,
    y: top,
    w: Math.max(1, Math.round(x + w) - left),
    h: Math.max(1, Math.round(y + h) - top),
  };
}
