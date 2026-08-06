import type { SpotMarket } from '@/libs/api/endpoints/marketApi.types';

/** Prefer exact symbol match from a spot search result list. */
export function pickSpotForSymbol(
  items: SpotMarket[] | undefined,
  symbol: string,
): SpotMarket | undefined {
  if (!items?.length || !symbol) return undefined;
  const target = symbol.trim().toUpperCase();
  const targetNorm = target.replace(/[-_/]/g, '');

  const exact =
    items.find((it) => (it.symbol ?? '').toUpperCase() === target) ??
    items.find(
      (it) => (it.symbol ?? '').toUpperCase().replace(/[-_/]/g, '') === targetNorm,
    );
  if (exact) return exact;

  return items.find((it) => {
    const s = (it.symbol ?? '').toUpperCase();
    return s === target || s.replace(/[-_/]/g, '') === targetNorm;
  });
}
