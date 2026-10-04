import type { LiqHeatVenue } from '@/components/organisms/LiquidationHeatmap';
import type { LiquidationOpenLevelBook, LiquidationOpenLevels } from '@/libs/api';

export type OpenLevelTableRow = {
  exchange: string;
  side: 'long' | 'short';
  leverage: number;
  price: number;
  notional: number;
  openedAt: string;
};

/** The book for the selected venue. Combined is withheld when any venue is missing. */
export function pickOpenLevelBook(
  data: LiquidationOpenLevels | undefined,
  venue: LiqHeatVenue,
): LiquidationOpenLevelBook | undefined {
  if (!data) return undefined;
  if (venue === 'binance') return data.binance;
  if (venue === 'bybit') return data.bybit;
  if (venue === 'okx') return data.okx;
  return data.combined;
}

/** Open bands only. A hit or an open-interest close is counted in the summary, not the table. */
export function openLevelTableRows(book: LiquidationOpenLevelBook | undefined): OpenLevelTableRow[] {
  if (!book || book.definite === false) return [];
  const rows: OpenLevelTableRow[] = [];
  for (const segment of book.segments ?? []) {
    if (segment.status !== 'open') continue;
    if (segment.side !== 'long' && segment.side !== 'short') continue;
    if (segment.price == null || segment.notional == null || !segment.openedAt) continue;
    if (!Number.isFinite(segment.price) || segment.price <= 0) continue;
    if (!Number.isFinite(segment.notional) || segment.notional <= 0) continue;
    rows.push({
      exchange: segment.exchange || book.exchange || '',
      side: segment.side,
      leverage: segment.leverage ?? 0,
      price: segment.price,
      notional: segment.notional,
      openedAt: segment.openedAt,
    });
  }
  return rows;
}
