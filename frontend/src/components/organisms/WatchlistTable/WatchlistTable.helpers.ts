import type { SpotMarket } from '@/libs/api';

function normalizeSymbol(raw: string): string {
  return raw.trim().toUpperCase().replace(/[-_/]/g, '');
}

/** Prefer exact symbol match from a spot search result list. */
export function pickSpotForSymbol(
  items: SpotMarket[] | undefined,
  symbol: string,
): SpotMarket | undefined {
  if (!items?.length || !symbol) return undefined;
  const target = symbol.trim().toUpperCase();
  const targetNorm = normalizeSymbol(symbol);

  const exact =
    items.find((it) => (it.symbol ?? '').toUpperCase() === target) ??
    items.find((it) => normalizeSymbol(it.symbol ?? '') === targetNorm);
  if (exact) return exact;

  // Avoid matching WBTC when looking for BTC: require full-symbol containment only as fallback.
  const loose = items.find((it) => {
    const s = (it.symbol ?? '').toUpperCase();
    return s === target || normalizeSymbol(s) === targetNorm;
  });
  return loose;
}
