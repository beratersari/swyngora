import { useListSpotMarketsQuery, type MarketExchange, type SpotMarket } from '@/libs/api';
import { Text } from '@/components/atoms/Text';
import { SpotMetricValue } from '@/components/molecules/SpotMetricValue';
import { formatSymbolDisplay, type SpotMetricDef } from '@/libs/utils';
import { DEFAULT_SPOT_POLL_MS } from '@/config/constants';
import { pickSpotForSymbol } from './WatchlistTable.helpers';

type PairProps = {
  symbol: string;
};

type MetricProps = {
  exchange: string;
  symbol: string;
  metric: SpotMetricDef;
};

/**
 * Live spot snapshot for one watchlist row.
 * RTK Query dedupes identical args across metric cells.
 */
export function useWatchlistSpot(exchange: string, symbol: string): {
  spot: SpotMarket | undefined;
  isLoading: boolean;
  isError: boolean;
} {
  const q = useListSpotMarketsQuery(
    {
      exchange: exchange as MarketExchange,
      q: symbol,
      limit: 25,
      status: 'TRADING',
      sort: 'quoteVolume',
      order: 'desc',
    },
    {
      skip: !exchange || !symbol,
      pollingInterval: DEFAULT_SPOT_POLL_MS,
      refetchOnFocus: true,
    },
  );
  const spot = pickSpotForSymbol(q.data?.items, symbol);
  const isLoading = Boolean(q.isLoading || (q.isFetching && !spot));
  return { spot, isLoading, isError: Boolean(q.isError) };
}

export function WatchlistPairCell({ symbol }: PairProps) {
  return (
    <Text variant="label" mono color="primary">
      {formatSymbolDisplay(symbol)}
    </Text>
  );
}

/** One metric cell driven by the shared catalog definition. */
export function WatchlistMetricCell({ exchange, symbol, metric }: MetricProps) {
  const { spot, isLoading } = useWatchlistSpot(exchange, symbol);
  return (
    <SpotMetricValue metric={metric} spot={spot} exchange={exchange} isLoading={isLoading} />
  );
}
