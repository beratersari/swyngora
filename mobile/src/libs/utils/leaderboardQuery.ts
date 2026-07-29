import type { MarketExchange, SpotListQuery, SpotMarket } from '@/libs/api';
import {
  LEADERBOARD_DEFAULT_EXCHANGE,
  LEADERBOARD_DEFAULT_QUOTE,
  LEADERBOARD_PAGE_SIZE,
  type LeaderboardKind,
} from '@/config/leaderboardConstants';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
} from './formatMarket';
import { formatPrice } from './formatPrice';
import type { MarketRowViewModel } from '@/components/organisms/market-row';

export type BuildLeaderboardSpotQueryInput = {
  board: LeaderboardKind;
  exchange?: MarketExchange | string;
  quote?: string;
  limit?: number;
  offset?: number;
};

/** Spot list args for a leaderboard board. */
export function buildLeaderboardSpotQuery(
  input: BuildLeaderboardSpotQueryInput,
): SpotListQuery {
  const exchange = (input.exchange as MarketExchange | undefined) ?? LEADERBOARD_DEFAULT_EXCHANGE;
  const quote = input.quote || LEADERBOARD_DEFAULT_QUOTE;
  const limit = input.limit ?? LEADERBOARD_PAGE_SIZE;
  const offset = input.offset ?? 0;

  let sort: SpotListQuery['sort'] = 'quoteVolume';
  let order: SpotListQuery['order'] = 'desc';

  if (input.board === 'gainers') {
    sort = 'priceChangePercent';
    order = 'desc';
  } else if (input.board === 'losers') {
    sort = 'priceChangePercent';
    order = 'asc';
  } else {
    sort = 'quoteVolume';
    order = 'desc';
  }

  return {
    exchange,
    quote,
    sort,
    order,
    limit,
    offset,
    status: 'TRADING',
  };
}

/** 1-based rank for infinite-scroll pages. */
export function rankLabel(offset: number, index: number): string {
  return `#${offset + index + 1}`;
}

export function mapSpotToLeaderboardRow(
  item: SpotMarket,
  rank: string,
): MarketRowViewModel {
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
    rankLabel: rank,
  };
}

export function mapSpotListToLeaderboardRows(
  items: SpotMarket[] | undefined,
  offset: number,
): MarketRowViewModel[] {
  return (items ?? []).map((item, i) =>
    mapSpotToLeaderboardRow(item, rankLabel(offset, i)),
  );
}
