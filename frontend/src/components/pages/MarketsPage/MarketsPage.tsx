import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Tag } from 'antd';
import { Text } from '@/components/atoms/Text';
import { ExchangeTabs } from '@/features/markets/components/ExchangeTabs';
import { MarketsToolbar } from '@/features/markets/components/MarketsToolbar';
import { MarketsTable } from '@/features/markets/components/MarketsTable';
import {
  rtkErrorMessage,
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  type MarketExchange,
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
import { McapHintAlert, MetaRow, PageIntro, PageStack } from './MarketsPage.styles';

/** Multi-exchange spot markets dashboard (Epic B). */
export function MarketsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
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

  const items = spotQuery.data?.items ?? [];
  const total = spotQuery.data?.total ?? 0;
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

  const onPageChange = (offset: number, limit: number) => {
    patchState({ offset, limit });
  };

  return (
    <PageStack>
      <PageIntro>
        <Text variant="h2" color="cream">
          Markets
        </Text>
        <Text variant="body" color="steel">
          Spot markets across Binance, Coinbase, and Bybit. Sort, filter, and page via the API —
          list refreshes every {DEFAULT_SPOT_POLL_MS / 1000}s while this tab is visible.
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
        {spotQuery.isFetching ? <Tag color="processing">updating…</Tag> : null}
        {spotQuery.isSuccess ? (
          <Tag color="success">
            {total.toLocaleString()} match{total === 1 ? '' : 'es'}
          </Tag>
        ) : null}
        {!visible ? <Tag>polling paused (tab hidden)</Tag> : null}
        <Text variant="caption" color="secondary">
          Default quote {DEFAULT_MARKETS_STATE.quote} · sort {state.sort} {state.order}
        </Text>
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
      />
    </PageStack>
  );
}
