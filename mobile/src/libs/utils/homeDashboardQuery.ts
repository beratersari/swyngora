import type { MarketExchange, SpotListQuery, SpotMarket } from '@/libs/api';
import {
  HOME_DEFAULT_EXCHANGE,
  HOME_DEFAULT_QUOTE,
  HOME_MOVERS_LIMIT,
  HOME_PUMP_TEASER_LIMIT,
  HOME_VOLUME_LIMIT,
} from '@/config/homeDashboardConstants';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
} from './formatMarket';
import { formatPrice } from './formatPrice';
import { formatPumpReturnPct, pumpReturnTone } from './formatPump';
import { buildLeaderboardSpotQuery } from './leaderboardQuery';
import { buildScanQuery, defaultPumpScanFilters } from './pumpQuery';
import { watchKey, type WatchlistPair } from './watchlistKey';

export type DashboardMarketRow = {
  id: string;
  exchange: string;
  symbol: string;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  metaLabel?: string;
};

export type DashboardPumpTeaser = {
  id: string;
  exchange: string;
  symbol: string;
  returnLabel: string;
  returnTone: 'success' | 'error' | 'secondary';
  metaLabel: string;
};

export function buildMoversSpotQuery(
  exchange: MarketExchange = HOME_DEFAULT_EXCHANGE,
  quote: string = HOME_DEFAULT_QUOTE,
  limit: number = HOME_MOVERS_LIMIT,
): SpotListQuery {
  return buildLeaderboardSpotQuery({
    board: 'gainers',
    exchange,
    quote,
    limit,
    offset: 0,
  });
}

export function buildVolumeSpotQuery(
  exchange: MarketExchange = HOME_DEFAULT_EXCHANGE,
  quote: string = HOME_DEFAULT_QUOTE,
  limit: number = HOME_VOLUME_LIMIT,
): SpotListQuery {
  return buildLeaderboardSpotQuery({
    board: 'volume',
    exchange,
    quote,
    limit,
    offset: 0,
  });
}

export function buildHomePumpScanQuery(
  exchange: MarketExchange = HOME_DEFAULT_EXCHANGE,
  symbolLimit: number = HOME_PUMP_TEASER_LIMIT,
) {
  const base = defaultPumpScanFilters(exchange);
  return buildScanQuery({ ...base, symbolLimit });
}

export function mapSpotToDashboardRow(
  item: SpotMarket,
  exchange: string,
): DashboardMarketRow {
  const symbol = item.symbol ?? '—';
  const ex = exchange;
  return {
    id: watchKey(ex, symbol),
    exchange: ex,
    symbol,
    lastPriceLabel: formatPrice(item.lastPrice),
    changePercentLabel: formatChangePercent(item.priceChangePercent),
    changeTone: changeTone(item.priceChangePercent),
    metaLabel: formatCompactUsd(item.quoteVolume),
  };
}

export function mapSpotListToDashboardRows(
  items: SpotMarket[] | undefined,
  exchange: string,
): DashboardMarketRow[] {
  return (items ?? []).map((i) => mapSpotToDashboardRow(i, exchange));
}

/** Favorites as dashboard rows; fill metrics from spot maps when available. */
export function mapFavoritesToDashboardRows(
  pairs: WatchlistPair[],
  limit: number,
  spotByKey?: Map<string, DashboardMarketRow>,
): DashboardMarketRow[] {
  return pairs.slice(0, limit).map((p) => {
    const key = watchKey(p.exchange, p.symbol);
    const hit = spotByKey?.get(key);
    if (hit) return hit;
    return {
      id: key,
      exchange: p.exchange,
      symbol: p.symbol,
      lastPriceLabel: '—',
      changePercentLabel: '—',
      changeTone: 'secondary' as const,
      metaLabel: p.exchange,
    };
  });
}

export function mapPumpHitsToTeasers(
  hits:
    | {
        exchange?: string;
        symbol?: string;
        bestReturnPct?: number;
        events?: unknown[];
        interval?: string;
      }[]
    | undefined,
  fallbackExchange: string,
): DashboardPumpTeaser[] {
  return (hits ?? []).map((h, i) => {
    const exchange = h.exchange ?? fallbackExchange;
    const symbol = h.symbol ?? '—';
    const ret = h.bestReturnPct;
    const eventCount = h.events?.length;
    return {
      id: `${watchKey(exchange, symbol)}-${i}`,
      exchange,
      symbol,
      returnLabel: formatPumpReturnPct(ret),
      returnTone: pumpReturnTone(ret),
      metaLabel: [h.interval, eventCount != null ? `${eventCount} evt` : '']
        .filter(Boolean)
        .join(' · '),
    };
  });
}

/** Index dashboard rows by exchange|symbol for favorite merge. */
export function indexDashboardRows(
  rows: DashboardMarketRow[],
): Map<string, DashboardMarketRow> {
  const map = new Map<string, DashboardMarketRow>();
  for (const r of rows) {
    map.set(watchKey(r.exchange, r.symbol), r);
  }
  return map;
}
