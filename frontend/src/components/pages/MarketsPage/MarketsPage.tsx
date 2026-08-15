import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Alert, Button, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { ExchangeTabs } from '@/components/organisms/ExchangeTabs';
import { MarketsToolbar } from '@/components/organisms/MarketsToolbar';
import { MarketsTable } from '@/components/organisms/MarketsTable';
import { getResultsRange } from '@/components/organisms/MarketsTable/MarketsTable.helpers';
import {
  rtkErrorMessage,
  useAddWatchlistItemMutation,
  useGetWatchlistQuery,
  useListExchangesQuery,
  useListProductTagsQuery,
  useListDelistScheduleQuery,
  useListSpotMarketsQuery,
  useRemoveWatchlistItemMutation,
  type MarketExchange,
  type SpotMarket,
  type SpotSortField,
  type SpotSortOrder,
} from '@/libs/api';
import { MetricColumnPicker } from '@/components/molecules/MetricColumnPicker';
import { useDebouncedValue, useDocumentVisible, useSpotMetricColumns } from '@/libs/hooks';
import { usePriceSubscription, useRealtimeConnected } from '@/libs/realtime';
import {
  defaultQuoteForExchange,
  marketsStateToSearchParams,
  metricColumnTitle,
  parseMarketsSearchParams,
  rtkCurrent,
  toSpotListQuery,
  type MarketsUrlState,
} from '@/libs/utils';
import { DEFAULT_SPOT_POLL_MS, SPOT_LIST_WS_REST_POLL_MS } from '@/config/constants';
import {
  McapHintAlert,
  MetaLeft,
  MetaRight,
  MetaRow,
  PageStack,
  ResultsBadge,
  ResultsCount,
  ResultsLabel,
} from './MarketsPage.styles';

/** Multi-exchange spot markets dashboard (Epic B). */
export function MarketsPage() {
  const { t } = useTranslation(['markets', 'common']);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const visible = useDocumentVisible();
  const metricColumns = useSpotMetricColumns('markets');

  const state = useMemo(() => parseMarketsSearchParams(searchParams), [searchParams]);
  const [qInput, setQInput] = useState(state.q);
  const debouncedQ = useDebouncedValue(qInput, 300);

  useEffect(() => {
    setQInput(state.q);
  }, [state.q]);

  /**
   * Functional URL updates — always merge from the latest search params so sort /
   * pagination / search debounce never overwrite each other with a stale `state`.
   */
  const patchState = useCallback(
    (patch: Partial<MarketsUrlState>) => {
      setSearchParams(
        (prev) => {
          const current = parseMarketsSearchParams(prev);
          return marketsStateToSearchParams({ ...current, ...patch });
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  // Search q is owned by the debounce effect only (not injected into every patch).
  useEffect(() => {
    setSearchParams(
      (prev) => {
        const current = parseMarketsSearchParams(prev);
        if (debouncedQ === current.q) return prev;
        return marketsStateToSearchParams({ ...current, q: debouncedQ, offset: 0 });
      },
      { replace: true },
    );
  }, [debouncedQ, setSearchParams]);

  const exchangesQuery = useListExchangesQuery();
  const tagsQuery = useListProductTagsQuery({ exchange: state.exchange });
  const spotArgs = useMemo(() => toSpotListQuery(state, debouncedQ), [state, debouncedQ]);
  const watchlistQuery = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const [addWatch] = useAddWatchlistItemMutation();
  const [removeWatch] = useRemoveWatchlistItemMutation();
  const livePrices = useRealtimeConnected();

  const spotQuery = useListSpotMarketsQuery(spotArgs, {
    // Fast poll when offline; slow REST while WS is live so mcap/tradeCount stay fresh.
    pollingInterval: !visible ? 0 : livePrices ? SPOT_LIST_WS_REST_POLL_MS : DEFAULT_SPOT_POLL_MS,
    refetchOnFocus: true,
  });
  const delistQuery = useListDelistScheduleQuery(
    { exchange: state.exchange },
    { skip: state.exchange !== 'binance' },
  );
  const delistEnabled = delistQuery.data?.enabled !== false;
  const delistCount = delistQuery.data?.items?.length ?? 0;

  const watchedKeys = useMemo(() => {
    const set = new Set<string>();
    for (const it of watchlistQuery.data?.items ?? []) {
      if (it.exchange && it.symbol) set.add(`${it.exchange}:${it.symbol}`);
    }
    return set;
  }, [watchlistQuery.data?.items]);

  const exchanges = exchangesQuery.data?.exchanges?.length
    ? exchangesQuery.data.exchanges
    : ['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'];

  // Keep last rows only for the *current* filter set (sort/page/poll).
  // Key the cache so a filter change never paints the previous venue's rows
  // for one frame before a useEffect can clear (stale flash).
  const filterKey = `${state.exchange}|${state.quote}|${state.tag}|${debouncedQ}`;
  const [rowCache, setRowCache] = useState<{
    key: string;
    items: SpotMarket[];
    total: number;
  }>({ key: filterKey, items: [], total: 0 });

  const liveSpot = rtkCurrent(spotQuery);
  useEffect(() => {
    if (liveSpot?.items) {
      setRowCache({
        key: filterKey,
        items: liveSpot.items,
        total: liveSpot.total ?? liveSpot.items.length,
      });
    }
  }, [liveSpot, filterKey]);

  const cachedForFilter = rowCache.key === filterKey ? rowCache : null;
  // currentData only — never paint the previous venue from RTK `.data`.
  const items = liveSpot?.items ?? cachedForFilter?.items ?? [];
  const liveSymbols = useMemo(
    () =>
      items
        .filter((it) => it.symbol)
        .map((it) => ({ exchange: state.exchange, symbol: String(it.symbol) })),
    [items, state.exchange],
  );
  usePriceSubscription(liveSymbols, visible);
  const total = liveSpot?.total ?? cachedForFilter?.total ?? 0;
  const hasRows = items.length > 0;
  const loadErrorText = spotQuery.isError
    ? rtkErrorMessage(spotQuery.error, {
        resource: t('markets:resource'),
        statusMessages: {
          502: t('markets:errors.upstreamMcap'),
        },
      })
    : null;
  // Full-table error only when there is nothing to show. Poll/refetch failures
  // keep last rows (RTK keeps data on reject) so sort/paging stay usable.
  const tableErrorMessage = hasRows ? null : loadErrorText;

  const onExchangeChange = (exchange: MarketExchange) => {
    // Reset quote to the venue's primary book (USDT vs USD) so Coinbase is not
    // left filtered to a handful of USDT pairs after browsing Binance.
    patchState({
      exchange,
      quote: defaultQuoteForExchange(exchange),
      offset: 0,
      tag: '',
    });
  };

  const onSortChange = (sort: SpotSortField, order: SpotSortOrder) => {
    patchState({ sort, order, offset: 0 });
  };

  const onPageChange = useCallback(
    (offset: number, limit: number) => {
      patchState({ offset, limit });
    },
    [patchState],
  );

  const range = getResultsRange(state.offset, state.limit, total);
  const rangeLabel =
    range.kind === 'empty'
      ? t('markets:results.emptyMatches')
      : t('markets:results.range', {
          from: range.from.toLocaleString(),
          to: range.to.toLocaleString(),
          total: range.total.toLocaleString(),
        });
  // Full-table skeleton only on first paint for a filter set (no rows yet).
  const isInitialLoading = items.length === 0 && (spotQuery.isLoading || spotQuery.isFetching);
  // In-table spinner for sort/page/poll refreshes while previous rows stay mounted.
  const isRefreshing = spotQuery.isFetching && items.length > 0;
  const pageNum = Math.floor(state.offset / state.limit) + 1;

  return (
    <PageStack>
      <PageHeader title={t('markets:title')} subtitle={t('markets:subtitle', { seconds: DEFAULT_SPOT_POLL_MS / 1000 })} />

      <ExchangeTabs
        exchanges={exchanges}
        value={state.exchange}
        onChange={onExchangeChange}
        isLoading={exchangesQuery.isLoading}
      />

      <MarketsToolbar
        q={qInput}
        quote={state.quote}
        tag={state.tag}
        tags={tagsQuery.data?.tags ?? []}
        tagsLoading={tagsQuery.isFetching}
        onQChange={(q) => setQInput(q)}
        onQuoteChange={(quote) => patchState({ quote, offset: 0 })}
        onTagChange={(tag) => patchState({ tag, offset: 0 })}
        trailing={
          <MetricColumnPicker
            available={metricColumns.available}
            value={metricColumns.metricIds}
            onChange={metricColumns.setMetricIds}
            onReset={metricColumns.resetToDefaults}
            getLabel={(key) => metricColumnTitle(t, key)}
            ariaLabel={t('markets:columns.aria')}
            buttonLabel={t('markets:columns.button')}
            resetLabel={t('markets:columns.reset')}
            moveUpLabel={t('markets:columns.moveUp')}
            moveDownLabel={t('markets:columns.moveDown')}
            dragHintLabel={t('markets:columns.dragHint')}
          />
        }
      />

      {state.exchange === 'binance' && delistQuery.isSuccess && !delistEnabled ? (
        <Alert
          type="info"
          showIcon
          message={t('markets:delist.disabledTitle')}
          description={t('markets:delist.disabledBody')}
        />
      ) : null}
      {state.exchange === 'binance' && delistQuery.isSuccess && delistEnabled && delistCount > 0 ? (
        <Text variant="caption" color="secondary">
          {t('markets:delist.activeHint', { count: delistCount })}
        </Text>
      ) : null}

      <MetaRow>
        <MetaLeft>
          {isInitialLoading ? (
            <ResultsBadge>
              <ResultsLabel>{t('markets:results.loading')}</ResultsLabel>
            </ResultsBadge>
          ) : spotQuery.isSuccess || items.length > 0 ? (
            <ResultsBadge>
              <ResultsCount>{total.toLocaleString()}</ResultsCount>
              <ResultsLabel>
                {t('markets:results.match', { count: total })}
                {total > 0 ? ` · ${rangeLabel}` : null}
              </ResultsLabel>
            </ResultsBadge>
          ) : spotQuery.isError ? (
            <ResultsBadge>
              <ResultsLabel>{t('markets:results.loadFailed')}</ResultsLabel>
            </ResultsBadge>
          ) : (
            <ResultsBadge>
              <ResultsLabel>{t('markets:results.noneYet')}</ResultsLabel>
            </ResultsBadge>
          )}
          {isRefreshing ? <BrandTag variant="live">{t('common:status.updating')}</BrandTag> : null}
          {!visible ? <BrandTag variant="paused">{t('common:status.pollingPaused')}</BrandTag> : null}
        </MetaLeft>
        <MetaRight>
          <Text variant="caption" color="secondary">
            {t('markets:results.meta', {
              quote: state.quote || defaultQuoteForExchange(state.exchange),
              sort: state.sort,
              order: state.order,
            })}
            {state.offset > 0
              ? ` · ${t('markets:results.page', { page: pageNum })}`
              : null}
          </Text>
        </MetaRight>
      </MetaRow>

      {spotQuery.isError && state.sort.startsWith('marketCap') ? (
        <McapHintAlert
          type="warning"
          showIcon
          message={t('markets:errors.mcapWarmupTitle')}
          description={t('markets:errors.mcapWarmupBody')}
        />
      ) : null}

      {loadErrorText && hasRows ? (
        <Alert
          type="warning"
          showIcon
          message={t('markets:errors.refreshFailed')}
          description={loadErrorText}
          action={
            <Button size="small" type="primary" onClick={() => void spotQuery.refetch()}>
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      <MarketsTable
        items={items}
        exchange={state.exchange}
        sort={state.sort}
        order={state.order}
        total={total}
        limit={state.limit}
        offset={state.offset}
        isLoading={isInitialLoading || isRefreshing}
        errorMessage={tableErrorMessage}
        onSortChange={onSortChange}
        onPageChange={onPageChange}
        onRetry={() => void spotQuery.refetch()}
        onRowOpen={(symbol) =>
          navigate(`/markets/${encodeURIComponent(state.exchange)}/${encodeURIComponent(symbol)}`)
        }
        watchedKeys={watchedKeys}
        onToggleWatch={(symbol, watched) => {
          const run = watched
            ? removeWatch({ exchange: state.exchange, symbol }).unwrap()
            : addWatch({ exchange: state.exchange, symbol }).unwrap();
          void run.catch((err) => {
            void message.error(
              rtkErrorMessage(err, { resource: t('markets:watchlistResource') }),
            );
          });
        }}
        metrics={metricColumns.metrics}
      />
    </PageStack>
  );
}
