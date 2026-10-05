import type { DerivativesCoin } from '@/libs/api';
import { financeColors, withAlpha } from '@/styles/tokens';
import { coinChartPath, signedAmount } from '@/components/organisms/DerivativesBoard/helpers';
import { HEATMAP_FULL, type HeatMetric } from './constants';
import type { HeatCell, HeatTone } from './DerivativesHeatmap.types';

const BED = '#E9EEF5';

type Reading = {
  value: number;
  neutral: number;
  text: string;
  heat: boolean;
};

export function heatmapCells(exchange: string, coins: DerivativesCoin[], metric: HeatMetric): HeatCell[] {
  const readings = coins.map((coin) => ({ coin, reading: readMetric(coin, metric) }));
  const full = scale(metric, readings.map((row) => row.reading));
  const cells = readings
    .filter((row) => row.coin.symbol)
    .map((row) => toCell(exchange, row.coin, row.reading, full));
  cells.sort((a, b) => {
    if (a.cell.tone === 'blank' && b.cell.tone !== 'blank') return 1;
    if (b.cell.tone === 'blank' && a.cell.tone !== 'blank') return -1;
    if (a.distance !== b.distance) return b.distance - a.distance;
    return a.cell.label.localeCompare(b.cell.label);
  });
  return cells.map((row) => row.cell);
}

export function heatFill(tone: HeatTone, strength: number): string {
  if (tone === 'blank' || tone === 'flat' || strength <= 0) return BED;
  const hex = tone === 'down' ? financeColors.down : tone === 'heat' ? financeColors.warning : financeColors.up;
  const alpha = 0.14 + 0.5 * Math.min(1, strength);
  return withAlpha(hex, alpha);
}

function scale(metric: HeatMetric, readings: (Reading | null)[]): number {
  const spec = HEATMAP_FULL[metric];
  if (spec !== 'relative') return spec;
  let max = 0;
  for (const reading of readings) {
    if (!reading) continue;
    max = Math.max(max, Math.abs(reading.value - reading.neutral));
  }
  return max > 0 ? max : 1;
}

function toCell(
  exchange: string,
  coin: DerivativesCoin,
  reading: Reading | null,
  full: number,
): { cell: HeatCell; distance: number } {
  const symbol = coin.symbol ?? '';
  const base = {
    symbol,
    label: coin.base || symbol,
    path: coinChartPath(exchange, symbol),
  };
  if (!reading) {
    return { cell: { ...base, text: '—', tone: 'blank', strength: 0 }, distance: -1 };
  }
  const distance = Math.abs(reading.value - reading.neutral);
  const strength = full > 0 ? Math.min(1, distance / full) : 0;
  let tone: HeatTone = 'flat';
  if (strength > 0) {
    if (reading.heat) tone = 'heat';
    else if (reading.value > reading.neutral) tone = 'up';
    else tone = 'down';
  }
  return { cell: { ...base, text: reading.text, tone, strength }, distance };
}

function readMetric(coin: DerivativesCoin, metric: HeatMetric): Reading | null {
  switch (metric) {
    case 'funding':
      return signed(coin.fundingDefinite, coin.fundingRatePct, 0, (raw) => `${raw}%`);
    case 'change1h':
      return signed(coin.change1hDefinite, coin.change1hPct, 0, (raw) => `${raw}%`);
    case 'change24h':
      return signed(coin.change24hDefinite, coin.change24hPct, 0, (raw) => `${raw}%`);
    case 'longPct':
      return signed(coin.longDefinite, coin.longPct, 50, (raw) => `${raw}%`);
    case 'takerBuyPct':
      return signed(coin.takerDefinite, coin.takerBuyPct, 50, (raw) => `${raw}%`);
    case 'gapPct':
      return signed(coin.gapDefinite, coin.gapPct, 0, (raw) => raw);
    case 'cvdDay':
      return signed(coin.cvdDayDefinite, coin.cvdDay, 0, (raw) => signedAmount(raw));
    case 'printChange':
      return signed(coin.printDefinite, coin.printChangePct, 0, (raw) => `${raw}%`);
    case 'orderCount':
      return signed(coin.orderDefinite, coin.orderRatio, 0, (raw) => raw);
    case 'unusual':
      return unusual(coin);
    case 'liqHeat':
      return heat(coin);
    case 'bidPct':
      return signed(coin.bidDefinite, coin.bidPct, 50, (raw) => `${raw}%`);
    default:
      return null;
  }
}

function signed(
  definite: boolean | undefined,
  raw: string | undefined,
  neutral: number,
  text: (raw: string) => string,
): Reading | null {
  if (!definite) return null;
  const value = finite(raw);
  if (value == null || raw == null) return null;
  return { value, neutral, text: text(raw), heat: false };
}

function unusual(coin: DerivativesCoin): Reading | null {
  if (!coin.unusualDefinite) return null;
  const distance = finite(coin.unusualDistance);
  if (distance == null) return null;
  const sign = unusualSign(coin.unusualBand);
  const body = (coin.unusualDistance ?? '').replace(/^[+-]/, '');
  const text = sign < 0 ? `-${body}` : sign > 0 ? `+${body}` : body;
  return { value: sign * Math.abs(distance), neutral: 0, text, heat: false };
}

function heat(coin: DerivativesCoin): Reading | null {
  if (!coin.liqHeatDefinite) return null;
  const value = finite(coin.liqHeatRatio);
  if (value == null) return null;
  return { value, neutral: 1, text: `${coin.liqHeatRatio}×`, heat: true };
}

function unusualSign(band: string | undefined): number {
  if (band === 'extreme_high' || band === 'high') return 1;
  if (band === 'extreme_low' || band === 'low') return -1;
  return 0;
}

function finite(value: string | undefined): number | null {
  if (value == null || value === '') return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}
