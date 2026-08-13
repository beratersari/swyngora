import type { HeatmapMetric, HeatmapTile, PriceChangeHeatmapItem } from './PriceChangeHeatmap.types';
import {
  HEATMAP_COLOR_CAP_PCT,
  HEATMAP_DEAD_ZONE_PCT,
  HEATMAP_GUTTER,
  HEATMAP_MAX_TILES,
  HEATMAP_NEUTRAL,
  HEATMAP_SCALE,
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

export function baseSymbol(symbol: string): string {
  return symbol.replace(/[-_]?(USDT|USDC|USD|BUSD|FDUSD)$/i, '') || symbol;
}

export function tileDensity(w: number, h: number): TileDensity {
  if (w >= 88 && h >= 68) return 'full';
  if (w >= 56 && h >= 40) return 'compact';
  if (w >= 36 && h >= 22) return 'ticker';
  return 'micro';
}

export function formatTileChange(changePct: number): string {
  if (!Number.isFinite(changePct)) return '0.0%';
  const sign = changePct > 0 ? '+' : '';
  return `${sign}${changePct.toFixed(1)}%`;
}

/** Opaque diverging fill. |Δ| inside the dead zone is true slate gray. */
export function changeFill(changePct: number): string {
  if (!Number.isFinite(changePct) || Math.abs(changePct) < HEATMAP_DEAD_ZONE_PCT) {
    return HEATMAP_NEUTRAL;
  }
  const clamped = Math.max(-HEATMAP_COLOR_CAP_PCT, Math.min(HEATMAP_COLOR_CAP_PCT, changePct));
  for (let i = 0; i < HEATMAP_SCALE.length - 1; i++) {
    const a = HEATMAP_SCALE[i]!;
    const b = HEATMAP_SCALE[i + 1]!;
    if (clamped <= b.at) {
      const span = b.at - a.at || 1;
      const t = (clamped - a.at) / span;
      return mixHex(a.fill, b.fill, t);
    }
  }
  return HEATMAP_SCALE[HEATMAP_SCALE.length - 1]!.fill;
}

function mixHex(a: string, b: string, t: number): string {
  const pa = parseHex(a);
  const pb = parseHex(b);
  const u = Math.min(1, Math.max(0, t));
  const r = Math.round(pa.r + (pb.r - pa.r) * u);
  const g = Math.round(pa.g + (pb.g - pa.g) * u);
  const bl = Math.round(pa.b + (pb.b - pa.b) * u);
  return `#${[r, g, bl].map((n) => n.toString(16).padStart(2, '0')).join('')}`;
}

function parseHex(s: string): { r: number; g: number; b: number } {
  const h = s.replace('#', '');
  if (h.length !== 6) return { r: 61, g: 68, b: 76 };
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
  return {
    ...tile,
    x: tile.x + g / 2,
    y: tile.y + g / 2,
    w: Math.max(1, tile.w - g),
    h: Math.max(1, tile.h - g),
  };
}
