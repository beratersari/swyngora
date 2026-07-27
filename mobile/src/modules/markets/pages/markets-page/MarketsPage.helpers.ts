import type { SpotMarket } from '@/libs/api';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
} from '@/libs/utils';
import type { MarketRowViewModel } from '@/components/organisms/market-row';

export function mapSpotMarketToRow(item: SpotMarket): MarketRowViewModel {
  const symbol = item.symbol ?? '—';
  return {
    id: symbol,
    symbol,
    lastPriceLabel: formatPrice(item.lastPrice),
    changePercentLabel: formatChangePercent(item.priceChangePercent),
    changeTone: changeTone(item.priceChangePercent),
    quoteVolumeLabel: formatCompactUsd(item.quoteVolume),
    marketCapLabel: formatCompactUsd(item.marketCapCirculating),
    tagsLabel: (item.tags ?? []).slice(0, 4).join(' · '),
  };
}

export function pageRangeLabel(offset: number, limit: number, total: number): string {
  if (total === 0) return '0 results';
  const start = offset + 1;
  const end = Math.min(offset + limit, total);
  return `${start}–${end} of ${total}`;
}

/** Favorites for a single exchange (case-insensitive). */
export function favoritesForExchange(
  items: { exchange: string; symbol: string }[],
  exchange: string,
): { exchange: string; symbol: string }[] {
  const ex = String(exchange).toLowerCase();
  return items.filter((i) => i.exchange.toLowerCase() === ex);
}

/**
 * When favorites-only is on, surface every favorite for the exchange.
 * Prefer loaded spot rows when present; otherwise placeholder metrics.
 */
export function buildFavoritesOnlyRows(
  favoritesOnExchange: { exchange: string; symbol: string }[],
  loadedRows: MarketRowViewModel[],
): MarketRowViewModel[] {
  const bySym = new Map(loadedRows.map((r) => [r.symbol.toUpperCase(), r]));
  return favoritesOnExchange.map((f) => {
    const existing = bySym.get(f.symbol.toUpperCase());
    if (existing) return existing;
    return {
      id: `${f.exchange}|${f.symbol}`,
      symbol: f.symbol,
      lastPriceLabel: '—',
      changePercentLabel: '—',
      changeTone: 'secondary',
      quoteVolumeLabel: '—',
      marketCapLabel: '—',
      tagsLabel: 'Favorite',
    };
  });
}

export function favoritesEmptyMessage(
  favoritesOnly: boolean,
  errorMessage: string | null,
  isLoading: boolean,
  displayCount: number,
): string | null {
  if (!favoritesOnly || errorMessage || isLoading || displayCount > 0) return null;
  return 'No favorites on this exchange yet — tap ★ on a row';
}

export function favoritesSummaryLabel(
  favoritesOnly: boolean,
  displayCount: number,
  defaultSummary: string | null,
): string | null {
  if (!favoritesOnly) return defaultSummary;
  return displayCount > 0 ? `Favorites: ${displayCount} shown` : null;
}
