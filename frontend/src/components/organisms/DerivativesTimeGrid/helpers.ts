import type { MarketDerivativesGridCell } from '@/libs/api';
import { coinChartPath } from '@/components/organisms/DerivativesBoard/helpers';
import { GRID_FULL, type GridMetric } from './constants';
import type { GridTone } from './DerivativesTimeGrid.types';

export type PaintedCell = {
  text: string;
  tone: GridTone;
  strength: number;
  title: string;
};

export function gridColumnLabel(iso: string, range: string, locale: string): string {
  const at = new Date(iso);
  if (Number.isNaN(at.getTime())) return iso;
  if (range === '7d') {
    return at.toLocaleDateString(locale, { month: 'short', day: 'numeric', timeZone: 'UTC' });
  }
  const hour = String(at.getUTCHours()).padStart(2, '0');
  const minute = String(at.getUTCMinutes()).padStart(2, '0');
  return `${hour}:${minute}`;
}

export function gridCoinPath(exchange: string, symbol: string): string {
  return coinChartPath(exchange, symbol);
}

export function paintGridCell(
  metric: GridMetric,
  cell: MarketDerivativesGridCell | undefined,
  cvdFull: number,
  blankTitle: string,
  missingTitle: (venues: string) => string,
): PaintedCell {
  if (!cell?.definite || cell.value == null || !cell.text) {
    const missing = cell?.missing?.filter(Boolean) ?? [];
    return {
      text: '—',
      tone: 'blank',
      strength: 0,
      title: missing.length > 0 ? missingTitle(missing.join(', ')) : blankTitle,
    };
  }
  const value = cell.value;
  if (metric === 'cvd') {
    const strength = cvdFull > 0 ? Math.min(1, Math.abs(value) / cvdFull) : 0;
    return { text: cell.text, tone: toneFromSign(value, strength), strength, title: cell.text };
  }
  if (metric === 'liquidations') {
    const heat = cell.heat;
    if (heat == null) {
      return { text: cell.text, tone: 'flat', strength: 0, title: cell.text };
    }
    const strength = Math.min(1, Math.max(0, heat - 1) / GRID_FULL.liquidations);
    return {
      text: cell.text,
      tone: strength > 0 ? 'heat' : 'flat',
      strength,
      title: cell.text,
    };
  }
  const full = metric === 'funding' ? GRID_FULL.funding : GRID_FULL.oi;
  const strength = Math.min(1, Math.abs(value) / full);
  return { text: cell.text, tone: toneFromSign(value, strength), strength, title: cell.text };
}

export function cvdScale(rows: { cells?: MarketDerivativesGridCell[] }[] | undefined): number {
  let max = 0;
  for (const row of rows ?? []) {
    for (const cell of row.cells ?? []) {
      if (cell.definite && cell.value != null) max = Math.max(max, Math.abs(cell.value));
    }
  }
  return max > 0 ? max : 1;
}

function toneFromSign(value: number, strength: number): GridTone {
  if (strength <= 0 || value === 0) return 'flat';
  return value > 0 ? 'up' : 'down';
}

export type GridDetailVenue = {
  exchange: string;
  text: string;
  definite: boolean;
};

/** Per-exchange rows for a cell click. A missing exchange is not a zero. */
export function gridDetailVenues(cell: MarketDerivativesGridCell | undefined): GridDetailVenue[] {
  const listed = cell?.venues?.filter((venue) => venue.exchange) ?? [];
  if (listed.length > 0) {
    return listed.map((venue) => {
      const definite = Boolean(venue.definite && venue.value != null && venue.text);
      return {
        exchange: venue.exchange ?? '',
        text: definite ? (venue.text ?? '') : '',
        definite,
      };
    });
  }
  const missing = cell?.missing?.filter(Boolean) ?? [];
  return missing.map((exchange) => ({ exchange, text: '', definite: false }));
}
