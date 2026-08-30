import { RSI_HEAT_NEUTRAL, RSI_HEAT_STOPS, RSI_HOVER_REACH, RSI_PLOT_PAD, RSI_TIP } from './constants';
import type { RSIHeatmapRow } from './RSIHeatmap.types';

/** Continuous fill from RSI 0–100. Missing values fall back to the zone, then ash. */
export function rsiFill(rsi: number | null | undefined, zone?: string | null): string {
  if (rsi != null && Number.isFinite(rsi)) {
    return lerpRsiFill(Math.min(100, Math.max(0, rsi)));
  }
  if (zone === 'oversold') return RSI_HEAT_STOPS[2]!.fill;
  if (zone === 'overbought') return RSI_HEAT_STOPS[6]!.fill;
  return RSI_HEAT_NEUTRAL;
}

function lerpRsiFill(rsi: number): string {
  const stops = RSI_HEAT_STOPS;
  if (rsi <= stops[0]!.at) return stops[0]!.fill;
  for (let i = 1; i < stops.length; i += 1) {
    const right = stops[i]!;
    if (rsi > right.at) continue;
    const left = stops[i - 1]!;
    const span = right.at - left.at;
    const t = span <= 0 ? 1 : (rsi - left.at) / span;
    if (t <= 0) return left.fill;
    if (t >= 1) return right.fill;
    return mixHex(left.fill, right.fill, t);
  }
  return stops[stops.length - 1]!.fill;
}

function mixHex(a: string, b: string, t: number): string {
  const pa = parseHex(a);
  const pb = parseHex(b);
  const u = Math.min(1, Math.max(0, t));
  const hex = (n: number) => Math.round(n).toString(16).padStart(2, '0');
  return `#${hex(pa.r + (pb.r - pa.r) * u)}${hex(pa.g + (pb.g - pa.g) * u)}${hex(pa.b + (pb.b - pa.b) * u)}`;
}

function parseHex(s: string): { r: number; g: number; b: number } {
  const h = s.replace('#', '');
  if (h.length !== 6) return { r: 161, g: 167, b: 187 };
  return {
    r: Number.parseInt(h.slice(0, 2), 16),
    g: Number.parseInt(h.slice(2, 4), 16),
    b: Number.parseInt(h.slice(4, 6), 16),
  };
}

export function formatRSI(rsi: number | null | undefined): string {
  if (rsi == null || Number.isNaN(rsi)) return '—';
  return rsi.toFixed(1);
}

export function rowLabel(row: RSIHeatmapRow): string {
  const base = (row.base ?? '').trim().toUpperCase();
  if (base) return base;
  return (row.symbol ?? '').trim().toUpperCase();
}

export function plottedRows(items: RSIHeatmapRow[]): RSIHeatmapRow[] {
  return items
    .filter((row) => row.rsi != null && Number.isFinite(row.rsi))
    .slice()
    .sort((a, b) => (a.rank ?? Number.POSITIVE_INFINITY) - (b.rank ?? Number.POSITIVE_INFINITY));
}

export function plotInner(width: number, height: number) {
  return {
    x: RSI_PLOT_PAD.left,
    y: RSI_PLOT_PAD.top,
    w: Math.max(1, width - RSI_PLOT_PAD.left - RSI_PLOT_PAD.right),
    h: Math.max(1, height - RSI_PLOT_PAD.top - RSI_PLOT_PAD.bottom),
  };
}

export function plotX(rank: number, count: number, width: number): number {
  const { x, w } = plotInner(width, 100);
  const n = Math.max(1, count);
  return x + ((Math.max(1, rank) - 0.5) / n) * w;
}

export function plotY(rsi: number, height: number): number {
  const { y, h } = plotInner(100, height);
  const clamped = Math.min(100, Math.max(0, rsi));
  return y + ((100 - clamped) / 100) * h;
}

export function rowKey(row: RSIHeatmapRow): string {
  return ((row.symbol || row.base || String(row.rank ?? '')).trim() || rowLabel(row)).toUpperCase();
}

/** Map a pointer from an element's CSS box into viewBox / plot coordinates. */
export function clientToPlot(
  rect: { left: number; top: number; width: number; height: number },
  clientX: number,
  clientY: number,
  viewW: number,
  viewH: number,
): { x: number; y: number } {
  const w = rect.width > 0 ? rect.width : 1;
  const h = rect.height > 0 ? rect.height : 1;
  return {
    x: ((clientX - rect.left) / w) * viewW,
    y: ((clientY - rect.top) / h) * viewH,
  };
}

/** Screen → viewBox via the SVG CTM so letterboxing does not pick the wrong coin. */
export function pointerToPlot(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
  viewW: number,
  viewH: number,
): { x: number; y: number } {
  const ctm = typeof svg.getScreenCTM === 'function' ? svg.getScreenCTM() : null;
  if (ctm && typeof svg.createSVGPoint === 'function') {
    const pt = svg.createSVGPoint();
    pt.x = clientX;
    pt.y = clientY;
    const mapped = pt.matrixTransform(ctm.inverse());
    if (Number.isFinite(mapped.x) && Number.isFinite(mapped.y)) {
      return { x: mapped.x, y: mapped.y };
    }
  }
  return clientToPlot(svg.getBoundingClientRect(), clientX, clientY, viewW, viewH);
}

export function tipOrigin(
  mx: number,
  my: number,
  frameW: number,
  frameH: number,
  tipW = RSI_TIP.w,
  tipH = RSI_TIP.h,
): { x: number; y: number } {
  const pad = 8;
  let x = mx + 16;
  let y = my + 16;
  if (x + tipW > frameW - pad) x = mx - tipW - 12;
  if (y + tipH > frameH - pad) y = my - tipH - 12;
  return { x: Math.max(pad, x), y: Math.max(pad, y) };
}

export function shouldLabelDot(row: RSIHeatmapRow, count: number): boolean {
  const rank = row.rank ?? 0;
  if (rank > 0 && rank <= 12) return true;
  if (row.zone === 'oversold' || row.zone === 'overbought') return true;
  return count <= 40 && rank <= 24;
}

export function nearestDot(
  rows: RSIHeatmapRow[],
  mx: number,
  my: number,
  width: number,
  height: number,
): RSIHeatmapRow | null {
  const reach2 = RSI_HOVER_REACH * RSI_HOVER_REACH;
  let best: RSIHeatmapRow | null = null;
  let bestScore = Number.POSITIVE_INFINITY;
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i]!;
    if (row.rsi == null) continue;
    const dx = plotX(i + 1, rows.length, width) - mx;
    const dy = plotY(row.rsi, height) - my;
    const d = dx * dx + dy * dy;
    if (d <= reach2 && d < bestScore) {
      bestScore = d;
      best = row;
    }
  }
  return best;
}
