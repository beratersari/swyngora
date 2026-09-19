import { useEffect, useMemo, useState } from 'react';
import {
  rtkErrorMessage,
  useGetTicker24hQuery,
  type MarketExchange,
  type Ticker24h,
} from '@/libs/api';
import {
  buildCrossExchangePlan,
  mapTickerToCrossExchangeRow,
  type CrossExchangePlanRow,
  type CrossExchangeRowModel,
} from '@/libs/utils';
import { CROSS_EXCHANGE_VENUES } from '@/config/crossExchangeConstants';
import { DETAIL_TICKER_POLL_MS } from './CoinDetailPage.constants';

function rtkHttpStatus(error: unknown): number | null {
  if (error && typeof error === 'object' && 'status' in error) {
    const s = (error as { status: unknown }).status;
    if (typeof s === 'number') return s;
  }
  return null;
}

function shouldTryNextCandidate(error: unknown): boolean {
  const st = rtkHttpStatus(error);
  return st === 404 || st === 400;
}

/**
 * One venue's ticker with candidate fallback (hooks must be called unconditionally).
 */
function useVenueCompareRow(
  exchange: MarketExchange,
  plan: CrossExchangePlanRow | undefined,
  opts: {
    sourceExchange: MarketExchange;
    sourceSymbol: string;
    sourceTicker: Ticker24h | undefined;
    sourceLoading: boolean;
    sourceError: unknown;
    sourceIsError: boolean;
    polling: boolean;
    skipAll: boolean;
  },
): { row: CrossExchangeRowModel; refetch: () => void } {
  const isSource = plan?.isSource ?? exchange === opts.sourceExchange;
  const candidates = plan?.candidates ?? [];
  const [idx, setIdx] = useState(0);

  const candidatesKey = candidates.join('|');
  useEffect(() => {
    setIdx(0);
  }, [candidatesKey, exchange, opts.sourceSymbol, opts.sourceExchange]);

  const safeIdx = candidates.length === 0 ? 0 : Math.min(idx, candidates.length - 1);
  const resolvedSymbol = candidates[safeIdx] ?? '';

  const query = useGetTicker24hQuery(
    { exchange, symbol: resolvedSymbol },
    {
      skip: opts.skipAll || isSource || !resolvedSymbol,
      pollingInterval:
        !opts.skipAll && !isSource && opts.polling ? DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  useEffect(() => {
    if (isSource || opts.skipAll || !query.isError) return;
    if (!shouldTryNextCandidate(query.error)) return;
    if (safeIdx >= candidates.length - 1) return;
    setIdx((i) => i + 1);
  }, [
    isSource,
    opts.skipAll,
    query.isError,
    query.error,
    safeIdx,
    candidates.length,
  ]);

  const row = useMemo(() => {
    const basePlan: CrossExchangePlanRow = plan ?? {
      exchange,
      candidates: resolvedSymbol ? [resolvedSymbol] : [],
      isSource,
    };

    if (isSource) {
      if (opts.sourceLoading && !opts.sourceTicker) {
        return mapTickerToCrossExchangeRow(basePlan, opts.sourceSymbol, undefined, {
          status: 'loading',
        });
      }
      if (opts.sourceIsError && !opts.sourceTicker) {
        return mapTickerToCrossExchangeRow(basePlan, opts.sourceSymbol, undefined, {
          status: 'error',
          errorMessage: rtkErrorMessage(opts.sourceError, { resource: 'ticker' }),
        });
      }
      if (opts.sourceTicker) {
        return mapTickerToCrossExchangeRow(
          basePlan,
          opts.sourceSymbol,
          opts.sourceTicker,
          { status: 'ok' },
        );
      }
      return mapTickerToCrossExchangeRow(basePlan, opts.sourceSymbol, undefined, {
        status: 'loading',
      });
    }

    if (!resolvedSymbol) {
      return mapTickerToCrossExchangeRow(basePlan, '—', undefined, {
        status: 'unavailable',
      });
    }

    if (query.data) {
      return mapTickerToCrossExchangeRow(basePlan, resolvedSymbol, query.data, {
        status: 'ok',
      });
    }

    if (query.isError) {
      const exhausted = safeIdx >= candidates.length - 1;
      return mapTickerToCrossExchangeRow(basePlan, resolvedSymbol, undefined, {
        status: exhausted ? 'unavailable' : 'loading',
        errorMessage: exhausted
          ? rtkErrorMessage(query.error, { resource: 'ticker' })
          : undefined,
      });
    }

    if (query.isLoading || query.isFetching || query.isUninitialized) {
      return mapTickerToCrossExchangeRow(basePlan, resolvedSymbol, undefined, {
        status: 'loading',
      });
    }

    return mapTickerToCrossExchangeRow(basePlan, resolvedSymbol, undefined, {
      status: 'unavailable',
    });
  }, [
    plan,
    exchange,
    isSource,
    resolvedSymbol,
    safeIdx,
    candidates.length,
    opts.sourceSymbol,
    opts.sourceTicker,
    opts.sourceLoading,
    opts.sourceError,
    opts.sourceIsError,
    query.isLoading,
    query.isFetching,
    query.isError,
    query.error,
    query.data,
  ]);

  return {
    row,
    refetch: () => {
      if (!isSource && resolvedSymbol) void query.refetch();
    },
  };
}

export type UseCrossExchangeRowsResult = {
  rows: CrossExchangeRowModel[];
  refetchAll: () => void;
};

export function useCrossExchangeRows(args: {
  sourceExchange: MarketExchange;
  sourceSymbol: string;
  sourceTicker: Ticker24h | undefined;
  sourceLoading: boolean;
  sourceError: unknown;
  sourceIsError: boolean;
  polling: boolean;
  skip: boolean;
}): UseCrossExchangeRowsResult {
  const plan = useMemo(
    () =>
      buildCrossExchangePlan(
        args.sourceExchange,
        args.sourceSymbol,
        CROSS_EXCHANGE_VENUES,
      ),
    [args.sourceExchange, args.sourceSymbol],
  );

  const planByExchange = useMemo(() => {
    const m = new Map<MarketExchange, CrossExchangePlanRow>();
    for (const p of plan) m.set(p.exchange, p);
    return m;
  }, [plan]);

  const shared = {
    sourceExchange: args.sourceExchange,
    sourceSymbol: args.sourceSymbol,
    sourceTicker: args.sourceTicker,
    sourceLoading: args.sourceLoading,
    sourceError: args.sourceError,
    sourceIsError: args.sourceIsError,
    polling: args.polling,
    skipAll: args.skip,
  };

  const binance = useVenueCompareRow(
    'binance',
    planByExchange.get('binance'),
    shared,
  );
  const coinbase = useVenueCompareRow(
    'coinbase',
    planByExchange.get('coinbase'),
    shared,
  );
  const bybit = useVenueCompareRow('bybit', planByExchange.get('bybit'), shared);

  const rows = useMemo(
    () => [binance.row, coinbase.row, bybit.row],
    [binance.row, coinbase.row, bybit.row],
  );

  const refetchAll = useMemo(
    () => () => {
      binance.refetch();
      coinbase.refetch();
      bybit.refetch();
    },
    [binance, coinbase, bybit],
  );

  return { rows, refetchAll };
}
