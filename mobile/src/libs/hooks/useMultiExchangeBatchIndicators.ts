import { useCallback, useMemo } from 'react';
import {
  rtkErrorMessage,
  usePostIndicatorsBatchQuery,
  type MarketExchange,
} from '@/libs/api';
import {
  batchItemKey,
  buildBatchIndicatorsArg,
  indexBatchItemsBySymbol,
  groupPairsByExchange,
  rsiFieldsFromItem,
  type RsiRowFields,
} from '@/libs/utils';
import {
  BATCH_FAVORITES_ENRICH_CAP,
  BATCH_INDICATORS_POLL_MS,
  BATCH_MAX_SYMBOLS,
} from '@/config/batchIndicatorsConstants';

const VENUES: MarketExchange[] = ['binance', 'coinbase', 'bybit'];

export type UseMultiExchangeBatchIndicatorsOptions = {
  /** When false, skip all batch calls (background / unfocused). */
  enabled: boolean;
  pollingIntervalMs?: number;
  enrichCap?: number;
};

/**
 * Up to one batch request per venue for favorites-style multi-exchange lists.
 * Hooks are fixed to three venues (rules of hooks).
 */
export function useMultiExchangeBatchIndicators(
  pairs: { exchange: string; symbol: string }[],
  options: UseMultiExchangeBatchIndicatorsOptions,
) {
  const enrichCap = options.enrichCap ?? BATCH_FAVORITES_ENRICH_CAP;
  const pollMs = options.pollingIntervalMs ?? BATCH_INDICATORS_POLL_MS;
  const enabled = options.enabled;

  const capped = useMemo(() => pairs.slice(0, enrichCap), [pairs, enrichCap]);
  const groups = useMemo(() => groupPairsByExchange(capped), [capped]);

  const binanceArg = useMemo(
    () =>
      buildBatchIndicatorsArg({
        exchange: 'binance',
        symbols: groups.binance.slice(0, BATCH_MAX_SYMBOLS),
      }),
    [groups.binance],
  );
  const coinbaseArg = useMemo(
    () =>
      buildBatchIndicatorsArg({
        exchange: 'coinbase',
        symbols: groups.coinbase.slice(0, BATCH_MAX_SYMBOLS),
      }),
    [groups.coinbase],
  );
  const bybitArg = useMemo(
    () =>
      buildBatchIndicatorsArg({
        exchange: 'bybit',
        symbols: groups.bybit.slice(0, BATCH_MAX_SYMBOLS),
      }),
    [groups.bybit],
  );

  const binanceQ = usePostIndicatorsBatchQuery(binanceArg, {
    skip: !enabled || binanceArg.symbols.length === 0,
    pollingInterval: enabled && binanceArg.symbols.length > 0 ? pollMs : 0,
    refetchOnFocus: false,
  });
  const coinbaseQ = usePostIndicatorsBatchQuery(coinbaseArg, {
    skip: !enabled || coinbaseArg.symbols.length === 0,
    pollingInterval: enabled && coinbaseArg.symbols.length > 0 ? pollMs : 0,
    refetchOnFocus: false,
  });
  const bybitQ = usePostIndicatorsBatchQuery(bybitArg, {
    skip: !enabled || bybitArg.symbols.length === 0,
    pollingInterval: enabled && bybitArg.symbols.length > 0 ? pollMs : 0,
    refetchOnFocus: false,
  });

  const queries = useMemo(
    () =>
      ({
        binance: binanceQ,
        coinbase: coinbaseQ,
        bybit: bybitQ,
      }) as const,
    [binanceQ, coinbaseQ, bybitQ],
  );

  const byKey = useMemo(() => {
    const map = new Map<string, RsiRowFields>();
    for (const ex of VENUES) {
      const q = queries[ex];
      const loading = q.isLoading || (q.isFetching && !q.data);
      const indexed = indexBatchItemsBySymbol(q.data?.items);
      const symbols = groups[ex] ?? [];
      for (const sym of symbols) {
        const item = indexed.get(sym.toUpperCase());
        map.set(batchItemKey(ex, sym), rsiFieldsFromItem(item, loading));
      }
    }
    return map;
  }, [queries, groups]);

  const isLoading =
    (binanceQ.isLoading || coinbaseQ.isLoading || bybitQ.isLoading) && byKey.size === 0;

  const errorMessage = useMemo(() => {
    for (const ex of VENUES) {
      const q = queries[ex];
      if (q.isError) {
        return rtkErrorMessage(q.error, { resource: 'indicators' });
      }
    }
    return null;
  }, [queries]);

  const disclaimer =
    binanceQ.data?.note ??
    coinbaseQ.data?.note ??
    bybitQ.data?.note ??
    null;

  const refetch = useCallback(() => {
    if (binanceArg.symbols.length) void binanceQ.refetch();
    if (coinbaseArg.symbols.length) void coinbaseQ.refetch();
    if (bybitArg.symbols.length) void bybitQ.refetch();
  }, [binanceArg.symbols.length, coinbaseArg.symbols.length, bybitArg.symbols.length, binanceQ, coinbaseQ, bybitQ]);

  const getRsiFields = useCallback(
    (exchange: string, symbol: string): RsiRowFields | undefined =>
      byKey.get(batchItemKey(exchange, symbol)),
    [byKey],
  );

  return {
    byKey,
    getRsiFields,
    isLoading,
    errorMessage,
    disclaimer,
    refetch,
  };
}
