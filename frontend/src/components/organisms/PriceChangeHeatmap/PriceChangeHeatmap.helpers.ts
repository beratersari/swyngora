import type { HeatmapMetric, HeatmapTile, PriceChangeHeatmapItem } from './PriceChangeHeatmap.types';
import {
  HEATMAP_BANDS,
  HEATMAP_GUTTER,
  HEATMAP_MAX_TILES,
  HEATMAP_NEUTRAL,
  HEATMAP_TILE_INK,
  HEATMAP_TILE_INK_ON_NEUTRAL,
} from './PriceChangeHeatmap.constants';

export type TileDensity = 'full' | 'compact' | 'ticker' | 'micro';

export function parseNum(value: string | number | null | undefined): number {
  if (value == null || value === '') return 0;
  const n = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function itemWeight(item: PriceChangeHeatmapItem, metric: HeatmapMetric): number {
  if (metric === 'marketCap') {
    const m = item.marketCapCirculating;
    return m != null && Number.isFinite(m) && m > 0 ? m : 0;
  }
  return Math.max(0, parseNum(item.quoteVolume));
}

export function baseSymbol(symbol: string, base?: string | null): string {
  const fromApi = (base ?? '').trim();
  if (fromApi) return fromApi;
  return symbol;
}

export function tileDensity(w: number, h: number): TileDensity {
  if (w >= 72 && h >= 52) return 'full';
  if (w >= 48 && h >= 32) return 'compact';
  if (w >= 28 && h >= 20) return 'ticker';
  return 'micro';
}

export function formatTileChange(changePct: number): string {
  if (!Number.isFinite(changePct)) return '0.00%';
  const sign = changePct > 0 ? '+' : '';
  return `${sign}${changePct.toFixed(2)}%`;
}

/** Discrete CoinMarketCap-style fill. Near-zero stays slate gray. */
export function changeFill(changePct: number): string {
  if (!Number.isFinite(changePct)) return HEATMAP_NEUTRAL;
  for (const band of HEATMAP_BANDS) {
    if (changePct <= band.upTo) return band.fill;
  }
  return HEATMAP_BANDS[HEATMAP_BANDS.length - 1]!.fill;
}

export function hoverCardOrigin(
  mouseX: number,
  mouseY: number,
  frameW: number,
  frameH: number,
  cardW = 210,
  cardH = 148,
): { x: number; y: number } {
  const pad = 10;
  let x = mouseX + 16;
  let y = mouseY + 16;
  if (x + cardW > frameW - pad) x = mouseX - cardW - 12;
  if (y + cardH > frameH - pad) y = mouseY - cardH - 12;
  return { x: Math.max(pad, x), y: Math.max(pad, y) };
}

export function tileInk(fill: string): string {
  if (fill === HEATMAP_NEUTRAL) return HEATMAP_TILE_INK_ON_NEUTRAL;
  const { r, g, b } = parseHex(fill);
  const lum = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
  return lum > 0.72 ? HEATMAP_TILE_INK_ON_NEUTRAL : HEATMAP_TILE_INK;
}

function parseHex(s: string): { r: number; g: number; b: number } {
  const h = s.replace('#', '');
  if (h.length !== 6) return { r: 161, g: 167, b: 187 };
  const n = Number.parseInt(h, 16);
  return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
}

type Rect = { x: number; y: number; w: number; h: number };
type Seed = Omit<HeatmapTile, 'x' | 'y' | 'w' | 'h'>;

export function toHeatmapTiles(
  items: PriceChangeHeatmapItem[],
  metric: HeatmapMetric,
  width: number,
  height: number,
): HeatmapTile[] {
  if (width <= 0 || height <= 0) return [];
  const ranked = items
    .filter((it) => it.symbol)
    .map((it) => ({
      symbol: it.symbol!,
      base: it.base,
      exchange: it.exchange,
      lastPrice: it.lastPrice,
      quoteVolume: it.quoteVolume,
      marketCapCirculating: it.marketCapCirculating,
      changePct: parseNum(it.priceChangePercent),
      weight: itemWeight(it, metric),
    }))
    .sort((a, b) => b.weight - a.weight)
    .slice(0, HEATMAP_MAX_TILES);
  const peak = ranked[0]?.weight ?? 0;
  const floor = peak > 0 ? peak * 0.002 : 1;
  const prepared: Seed[] = ranked.map((it) => ({
    ...it,
    weight: Math.max(it.weight, floor),
  }));
  if (!prepared.length) return [];
  return squarify(prepared, { x: 0, y: 0, w: width, h: height });
}

/**
 * Squarified treemap (Bruls / d3-hierarchy) so cells stay closer to square
 * and can hold centered labels.
 */
export function squarify(items: Seed[], rect: Rect): HeatmapTile[] {
  const nodes = items.slice();
  let remaining = nodes.reduce((s, it) => s + it.weight, 0);
  if (remaining <= 0) return [];
  let { x, y, w, h } = rect;
  const out: HeatmapTile[] = [];
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
    // Row sits against the shortest side so cells stay closer to square.
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

function insetTile(tile: HeatmapTile): HeatmapTile {
  const g = HEATMAP_GUTTER;
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
