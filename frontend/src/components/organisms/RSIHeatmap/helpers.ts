import { RSI_DOT_RADIUS, RSI_HEAT_BANDS, RSI_HEAT_NEUTRAL, RSI_PLOT_PAD } from './constants';
import type { RSIHeatmapRow } from './RSIHeatmap.types';

/** Color from the API `zone` field (Go RSIZoneFor). */
export function rsiFill(zone: string | null | undefined): string {
  if (zone === 'oversold') return RSI_HEAT_BANDS[1]!.fill;
  if (zone === 'overbought') return RSI_HEAT_BANDS[7]!.fill;
  return RSI_HEAT_NEUTRAL;
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
  const reach = RSI_DOT_RADIUS + 8;
  let best: RSIHeatmapRow | null = null;
  let bestScore = Number.POSITIVE_INFINITY;
  for (const row of rows) {
    if (row.rsi == null) continue;
    const dx = plotX(row.rank ?? 0, rows.length, width) - mx;
    const dy = plotY(row.rsi, height) - my;
    const d = dx * dx + dy * dy;
    if (d <= reach * reach && d < bestScore) {
      bestScore = d;
      best = row;
    }
  }
  return best;
}
