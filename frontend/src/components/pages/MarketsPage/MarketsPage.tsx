import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Tag } from 'antd';
import { Text } from '@/components/atoms/Text';
import { ExchangeTabs } from '@/components/organisms/ExchangeTabs';
import { MarketsToolbar } from '@/components/organisms/MarketsToolbar';
import { MarketsTable } from '@/components/organisms/MarketsTable';
import { formatResultsRange } from '@/components/organisms/MarketsTable/MarketsTable.helpers';
import {
  rtkErrorMessage,
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  type MarketExchange,
  type SpotMarket,
  type SpotSortField,
  type SpotSortOrder,
} from '@/libs/api';
import { useDebouncedValue, useDocumentVisible } from '@/libs/hooks';
import {
  DEFAULT_MARKETS_STATE,
  marketsStateToSearchParams,
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
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const visible = useDocumentVisible();

  const state = useMemo(() => parseMarketsSearchParams(searchParams), [searchParams]);
  const [qInput, setQInput] = useState(state.q);
  const debouncedQ = useDebouncedValue(qInput, 300);

  useEffect(() => {
    setQInput(state.q);
  }, [state.q]);

  const patchState = useCallback(
    (patch: Partial<MarketsUrlState>) => {
      const next: MarketsUrlState = { ...state, ...patch };
      if (patch.q === undefined) {
        next.q = qInput;
      }
      setSearchParams(marketsStateToSearchParams(next), { replace: true });
    },
    [state, qInput, setSearchParams],
  );

  useEffect(() => {
    if (debouncedQ === state.q) return;
    const next = { ...state, q: debouncedQ, offset: 0 };
    setSearchParams(marketsStateToSearchParams(next), { replace: true });
  }, [debouncedQ, state, setSearchParams]);

  const exchangesQuery = useListExchangesQuery();
  const tagsQuery = useListProductTagsQuery({ exchange: state.exchange });
  const spotArgs = useMemo(() => toSpotListQuery(state, debouncedQ), [state, debouncedQ]);

  const spotQuery = useListSpotMarketsQuery(spotArgs, {
    pollingInterval: visible ? DEFAULT_SPOT_POLL_MS : 0,
    refetchOnFocus: true,
  });

  const exchanges = exchangesQuery.data?.exchanges?.length
    ? exchangesQuery.data.exchanges
    : ['binance', 'coinbase', 'bybit'];

  // Hold last successful page while the next offset loads (smoother paging only)
  const [cachedItems, setCachedItems] = useState<SpotMarket[]>([]);
  const [cachedTotal, setCachedTotal] = useState(0);
  const filterKey = `${state.exchange}|${state.quote}|${state.tag}|${debouncedQ}|${state.sort}|${state.order}`;

  useEffect(() => {
    // Drop stale rows when filters/sort change (not when only paging)
    setCachedItems([]);
    setCachedTotal(0);
  }, [filterKey]);

  useEffect(() => {
    if (spotQuery.data?.items) {
      setCachedItems(spotQuery.data.items);
      setCachedTotal(spotQuery.data.total ?? spotQuery.data.items.length);
    }
  }, [spotQuery.data]);

  const items = spotQuery.data?.items ?? cachedItems;
  const total = spotQuery.data?.total ?? cachedTotal;
  const errorMessage = spotQuery.isError
    ? rtkErrorMessage(spotQuery.error, {
        resource: 'markets',
        statusMessages: {
          502: 'Upstream or supply snapshot unavailable. Market-cap sorts need a warm supply cache — try quote volume sort or retry shortly.',
        },
      })
    : null;

  const onExchangeChange = (exchange: MarketExchange) => {
    patchState({ exchange, offset: 0, tag: '' });
  };

  const onSortChange = (sort: SpotSortField, order: SpotSortOrder) => {
    patchState({ sort, order, offset: 0 });
  };

  const onPageChange = useCallback(
    (offset: number, limit: number) => {
      // Always write offset/limit so page 2+ is reflected in the URL
      const next: MarketsUrlState = {
        ...state,
        q: qInput,
        offset,
        limit,
      };
      setSearchParams(marketsStateToSearchParams(next), { replace: true });
    },
    [state, qInput, setSearchParams],
  );

  const rangeLabel = formatResultsRange(state.offset, state.limit, total);
  const isInitialLoading = spotQuery.isLoading && items.length === 0;
  const isRefreshing = spotQuery.isFetching && items.length > 0;

  return (
    <PageStack>
      <PageIntro>
        <Text variant="h2" color="primary">
          Markets
        </Text>
        <Text variant="body" color="secondary">
          Spot markets across Binance, Coinbase, and Bybit. Sort, filter, and page via the API —
          list refreshes every {DEFAULT_SPOT_POLL_MS / 1000}s while this tab is visible. Click a row
          for chart and indicators.
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
      />

      <MetaRow>
        <MetaLeft>
          {isInitialLoading ? (
            <ResultsBadge>
              <ResultsLabel>Loading markets…</ResultsLabel>
            </ResultsBadge>
          ) : spotQuery.isSuccess || items.length > 0 ? (
            <ResultsBadge>
              <ResultsCount>{total.toLocaleString()}</ResultsCount>
              <ResultsLabel>
                {total === 1 ? 'match' : 'matches'}
                {total > 0 ? ` · ${rangeLabel}` : null}
              </ResultsLabel>
            </ResultsBadge>
          ) : spotQuery.isError ? (
            <ResultsBadge>
              <ResultsLabel>Could not load matches</ResultsLabel>
            </ResultsBadge>
          ) : (
            <ResultsBadge>
              <ResultsLabel>No results yet</ResultsLabel>
            </ResultsBadge>
          )}
          {isRefreshing ? <Tag color="processing">updating…</Tag> : null}
          {!visible ? <Tag color="default">polling paused</Tag> : null}
        </MetaLeft>
        <MetaRight>
          <Text variant="caption" color="secondary">
            Quote {state.quote || DEFAULT_MARKETS_STATE.quote} · sort {state.sort} {state.order}
            {state.offset > 0 ? ` · page ${Math.floor(state.offset / state.limit) + 1}` : null}
          </Text>
        </MetaRight>
      </MetaRow>

      {spotQuery.isError && state.sort.startsWith('marketCap') ? (
        <McapHintAlert
          type="warning"
          showIcon
          message="Market-cap data may be warming up"
          description="If the supply snapshot is empty, mcap sorts can fail. Switch sort to Quote volume or retry."
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
        isLoading={spotQuery.isLoading || spotQuery.isFetching}
        errorMessage={errorMessage}
        onSortChange={onSortChange}
        onPageChange={onPageChange}
        onRetry={() => void spotQuery.refetch()}
        onRowOpen={(symbol) =>
          navigate(`/markets/${encodeURIComponent(state.exchange)}/${encodeURIComponent(symbol)}`)
        }
      />
    </PageStack>
  );
}
