import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Alert, Button, Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
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
  useListSpotMarketsQuery,
  useRemoveWatchlistItemMutation,
  type MarketExchange,
  type SpotMarket,
  type SpotSortField,
  type SpotSortOrder,
} from '@/libs/api';
import { MetricColumnPicker } from '@/components/molecules/MetricColumnPicker';
import { useDebouncedValue, useDocumentVisible, useSpotMetricColumns } from '@/libs/hooks';
import {
  defaultQuoteForExchange,
  marketsStateToSearchParams,
  metricColumnTitle,
  parseMarketsSearchParams,
  toSpotListQuery,
  type MarketsUrlState,
} from '@/libs/utils';
import { DEFAULT_SPOT_POLL_MS } from '@/config/constants';
import {
  McapHintAlert,
  MetaLeft,
  MetaRight,
  MetaRow,
  PageIntro,
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
  const watchlistQuery = useGetWatchlistQuery();
  const [addWatch] = useAddWatchlistItemMutation();
  const [removeWatch] = useRemoveWatchlistItemMutation();

  const spotQuery = useListSpotMarketsQuery(spotArgs, {
    pollingInterval: visible ? DEFAULT_SPOT_POLL_MS : 0,
    refetchOnFocus: true,
  });

  const watchedKeys = useMemo(() => {
    const set = new Set<string>();
    for (const it of watchlistQuery.data?.items ?? []) {
      if (it.exchange && it.symbol) set.add(`${it.exchange}:${it.symbol}`);
    }
    return set;
  }, [watchlistQuery.data?.items]);

  const exchanges = exchangesQuery.data?.exchanges?.length
    ? exchangesQuery.data.exchanges
    : ['binance', 'coinbase', 'bybit'];

  // Keep last rows only for the *current* filter set (sort/page/poll).
  // Key the cache so a filter change never paints the previous venue's rows
  // for one frame before a useEffect can clear (stale flash).
  const filterKey = `${state.exchange}|${state.quote}|${state.tag}|${debouncedQ}`;
  const [rowCache, setRowCache] = useState<{
    key: string;
    items: SpotMarket[];
    total: number;
  }>({ key: filterKey, items: [], total: 0 });

  useEffect(() => {
    if (spotQuery.data?.items) {
      setRowCache({
        key: filterKey,
        items: spotQuery.data.items,
        total: spotQuery.data.total ?? spotQuery.data.items.length,
      });
    }
  }, [spotQuery.data, filterKey]);

  const cachedForFilter = rowCache.key === filterKey ? rowCache : null;
  // Prefer current query data; while a new sort/page is loading keep last rows.
  const items = spotQuery.data?.items ?? cachedForFilter?.items ?? [];
  const total = spotQuery.data?.total ?? cachedForFilter?.total ?? 0;
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
      <PageIntro>
        <Text variant="h2" color="primary">
          {t('markets:title')}
        </Text>
        <Text variant="body" color="secondary">
          {t('markets:subtitle', { seconds: DEFAULT_SPOT_POLL_MS / 1000 })}
        </Text>
      </PageIntro>

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
          {isRefreshing ? <Tag color="processing">{t('common:status.updating')}</Tag> : null}
          {!visible ? <Tag color="default">{t('common:status.pollingPaused')}</Tag> : null}
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
          if (watched) {
            void removeWatch({ exchange: state.exchange, symbol });
          } else {
            void addWatch({ exchange: state.exchange, symbol });
          }
        }}
        metrics={metricColumns.metrics}
      />
    </PageStack>
  );
}
