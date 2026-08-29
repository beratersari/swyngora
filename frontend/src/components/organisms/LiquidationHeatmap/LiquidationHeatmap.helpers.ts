import {
  AXIS_INK,
  GRID_LINE,
  HEAT_BED,
  HEAT_HIGH,
  HEAT_LOW,
  HEAT_MID,
  HEAT_PEAK,
  HEATMAP_PAD,
  LAST_STROKE,
  LONG_HIGH,
  LONG_LOW,
  PLOT_BG,
  SHORT_HIGH,
  SHORT_LOW,
} from './LiquidationHeatmap.constants';
import type {
  LiqHeatHover,
  LiqHeatLayout,
  LiqHeatSide,
  LiqHeatVenue,
  LiquidationHeatmapData,
  LiquidationHeatmapGrid,
  LiquidationHeatmapReviewVenue,
} from './LiquidationHeatmap.types';

type Rgb = { r: number; g: number; b: number };

export function pickGrid(
  data: LiquidationHeatmapData | undefined,
  venue: LiqHeatVenue,
): LiquidationHeatmapGrid | undefined {
  if (!data) return undefined;
  if (venue === 'binance') return data.binance;
  if (venue === 'bybit') return data.bybit;
  return data.combined;
}

export function pickReview(
  data: LiquidationHeatmapData | undefined,
  venue: LiqHeatVenue,
): LiquidationHeatmapReviewVenue | undefined {
  const review = data?.review;
  if (!review) return undefined;
  if (venue === 'binance') return review.binance;
  if (venue === 'bybit') return review.bybit;
  return review.combined;
}

export function formatHitRate(n: number | undefined): string {
  if (!Number.isFinite(n)) return '—';
  return `${Math.round((n ?? 0) * 100)}%`;
}

export function formatLookahead(sec: number | undefined): string {
  if (!Number.isFinite(sec) || !sec || sec <= 0) return '—';
  if (sec < 60) return `${Math.round(sec)}s`;
  if (sec < 3600) return `${Math.round(sec / 60)}m`;
  const hours = Math.floor(sec / 3600);
  const minutes = Math.round((sec % 3600) / 60);
  return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
}

export function pickMatrix(
  grid: LiquidationHeatmapGrid | undefined,
  side: LiqHeatSide,
): number[][] {
  if (!grid) return [];
  if (side === 'longs') return grid.longs ?? [];
  if (side === 'shorts') return grid.shorts ?? [];
  return grid.totals ?? [];
}

export function parseTimeMs(raw: string | undefined): number {
  const ms = Date.parse(raw ?? '');
  return Number.isFinite(ms) ? ms : NaN;
}

export function intensityFromNotional(notional: number, max: number): number {
  if (!(max > 0) || !(notional > 0)) return 0;
  const t = Math.log10(1 + notional) / Math.log10(1 + max);
  return Math.min(1, Math.max(0, t));
}

function mixRgb(a: Rgb, b: Rgb, t: number): Rgb {
  const u = Math.min(1, Math.max(0, t));
  return {
    r: Math.round(a.r + (b.r - a.r) * u),
    g: Math.round(a.g + (b.g - a.g) * u),
    b: Math.round(a.b + (b.b - a.b) * u),
  };
}

function rgbString(c: Rgb): string {
  return `rgb(${c.r}, ${c.g}, ${c.b})`;
}

/** CoinGlass-style gold → orange → magenta, or long/short tints. */
export function heatColor(intensity: number, side: LiqHeatSide): string {
  if (intensity <= 0) return rgbString(HEAT_BED);
  const t = Math.min(1, Math.max(0, intensity));
  if (side === 'longs') {
    return rgbString(mixRgb(LONG_LOW, LONG_HIGH, t));
  }
  if (side === 'shorts') {
    return rgbString(mixRgb(SHORT_LOW, SHORT_HIGH, t));
  }
  if (t < 0.35) {
    return rgbString(mixRgb(HEAT_LOW, HEAT_MID, t / 0.35));
  }
  if (t < 0.75) {
    return rgbString(mixRgb(HEAT_MID, HEAT_HIGH, (t - 0.35) / 0.4));
  }
  return rgbString(mixRgb(HEAT_HIGH, HEAT_PEAK, (t - 0.75) / 0.25));
}

export function cellValue(matrix: number[][], timeIdx: number, priceIdx: number): number {
  const col = matrix[timeIdx];
  if (!col) return 0;
  const v = col[priceIdx];
  return Number.isFinite(v) && v > 0 ? v : 0;
}

export function hasHeatTape(data: LiquidationHeatmapData | undefined): boolean {
  return Boolean(data?.times?.length && data.prices?.length);
}

export function buildLayout(
  data: LiquidationHeatmapData | undefined,
  width: number,
  height: number,
): LiqHeatLayout | null {
  const times = (data?.times ?? []).map((t) => parseTimeMs(t));
  const prices = (data?.prices ?? []).filter((p) => Number.isFinite(p) && p > 0);
  if (times.length === 0 || prices.length === 0) return null;
  const plotX = HEATMAP_PAD.left;
  const plotY = HEATMAP_PAD.top;
  const plotW = Math.max(8, width - HEATMAP_PAD.left - HEATMAP_PAD.right);
  const plotH = Math.max(8, height - HEATMAP_PAD.top - HEATMAP_PAD.bottom);
  return {
    plotX,
    plotY,
    plotW,
    plotH,
    cellW: plotW / times.length,
    cellH: plotH / prices.length,
    nT: times.length,
    nP: prices.length,
    times,
    prices,
  };
}

export function formatLiqPrice(price: number, step: number): string {
  if (!Number.isFinite(price)) return '—';
  if (step >= 1) return price.toFixed(0);
  if (step >= 0.1) return price.toFixed(1);
  if (step >= 0.01) return price.toFixed(2);
  if (Math.abs(price) >= 1000) return price.toFixed(0);
  if (Math.abs(price) >= 1) return price.toFixed(2);
  return price.toPrecision(4);
}

export function formatLiqNotional(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '—';
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  if (n >= 10) return n.toFixed(0);
  return n.toFixed(2);
}

export function formatLiqTime(timeMs: number, range?: string): string {
  if (!Number.isFinite(timeMs)) return '—';
  const d = new Date(timeMs);
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  if (range === '3d' || range === '7d') {
    const mo = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${mo}-${day} ${hh}:${mm}`;
  }
  return `${hh}:${mm}`;
}

export function yTickIndexes(nP: number): number[] {
  if (nP <= 0) return [];
  if (nP <= 6) return Array.from({ length: nP }, (_, i) => i);
  const out = [0];
  for (let k = 1; k < 5; k += 1) {
    out.push(Math.round((k * (nP - 1)) / 5));
  }
  out.push(nP - 1);
  return [...new Set(out)];
}

export function xTickIndexes(nT: number): number[] {
  if (nT <= 0) return [];
  if (nT <= 6) return Array.from({ length: nT }, (_, i) => i);
  const out = [0];
  for (let k = 1; k < 5; k += 1) {
    out.push(Math.round((k * (nT - 1)) / 5));
  }
  out.push(nT - 1);
  return [...new Set(out)];
}

export function hitTest(
  px: number,
  py: number,
  data: LiquidationHeatmapData | undefined,
  layout: LiqHeatLayout,
  venue: LiqHeatVenue,
): LiqHeatHover | null {
  const col = Math.floor((px - layout.plotX) / layout.cellW);
  const row = Math.floor((py - layout.plotY) / layout.cellH);
  if (col < 0 || col >= layout.nT || row < 0 || row >= layout.nP) return null;
  const grid = pickGrid(data, venue);
  return {
    x: px,
    y: py,
    timeMs: layout.times[col] ?? NaN,
    price: layout.prices[row] ?? NaN,
    longs: cellValue(grid?.longs ?? [], col, row),
    shorts: cellValue(grid?.shorts ?? [], col, row),
    totals: cellValue(grid?.totals ?? [], col, row),
  };
}

export function drawHeatmap(
  ctx: CanvasRenderingContext2D,
  data: LiquidationHeatmapData | undefined,
  layout: LiqHeatLayout,
  venue: LiqHeatVenue,
  side: LiqHeatSide,
  lastPrice: number | undefined,
  range: string,
): void {
  const { plotX, plotY, plotW, plotH, cellW, cellH, nT, nP, prices } = layout;
  ctx.fillStyle = PLOT_BG;
  ctx.fillRect(0, 0, plotX + plotW + HEATMAP_PAD.right, plotY + plotH + HEATMAP_PAD.bottom);

  const grid = pickGrid(data, venue);
  const matrix = pickMatrix(grid, side);
  const peak = grid?.maxIntensity ?? 0;

  for (let i = 0; i < nT; i += 1) {
    for (let j = 0; j < nP; j += 1) {
      const v = cellValue(matrix, i, j);
      if (v <= 0) continue;
      ctx.fillStyle = heatColor(intensityFromNotional(v, peak), side);
      ctx.fillRect(
        plotX + i * cellW,
        plotY + j * cellH,
        Math.max(1, cellW + 0.4),
        Math.max(1, cellH + 0.4),
      );
    }
  }

  ctx.strokeStyle = GRID_LINE;
  ctx.lineWidth = 1;
  for (const j of yTickIndexes(nP)) {
    const y = plotY + j * cellH + cellH / 2;
    ctx.beginPath();
    ctx.moveTo(plotX, y);
    ctx.lineTo(plotX + plotW, y);
    ctx.stroke();
  }

  if (lastPrice && lastPrice > 0 && prices.length > 1) {
    const pMax = prices[0] ?? 0;
    const pMin = prices[prices.length - 1] ?? 0;
    if (pMax > pMin && lastPrice >= pMin && lastPrice <= pMax) {
      const y = plotY + ((pMax - lastPrice) / (pMax - pMin)) * plotH;
      ctx.strokeStyle = LAST_STROKE;
      ctx.setLineDash([4, 3]);
      ctx.beginPath();
      ctx.moveTo(plotX, y);
      ctx.lineTo(plotX + plotW, y);
      ctx.stroke();
      ctx.setLineDash([]);
    }
  }

  const step = data?.priceStep ?? 0;
  ctx.fillStyle = AXIS_INK;
  ctx.font = '11px Inter, "Segoe UI", system-ui, sans-serif';
  ctx.textAlign = 'right';
  ctx.textBaseline = 'middle';
  for (const j of yTickIndexes(nP)) {
    const y = plotY + j * cellH + cellH / 2;
    ctx.fillText(formatLiqPrice(prices[j] ?? 0, step), plotX - 8, y);
  }
  ctx.textAlign = 'center';
  ctx.textBaseline = 'top';
  for (const i of xTickIndexes(nT)) {
    const x = plotX + i * cellW + cellW / 2;
    ctx.fillText(formatLiqTime(layout.times[i] ?? NaN, range), x, plotY + plotH + 6);
  }
}
