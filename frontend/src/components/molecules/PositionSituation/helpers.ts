import type { SpotPosition } from '@/libs/api';

const CASH_EQUITIES = new Set(['nasdaq', 'bist']);

export function isCashEquity(exchange: string | undefined): boolean {
  return CASH_EQUITIES.has((exchange ?? '').trim().toLowerCase());
}

/** The open spot row for this coin, when the selected book still holds it. */
export function openPositionFor(
  positions: SpotPosition[] | undefined,
  exchange: string,
  symbol: string,
): SpotPosition | undefined {
  return positions?.find(
    (row) =>
      row.exchange === exchange &&
      row.symbol === symbol &&
      (row.quantity ?? 0) > 0,
  );
}
