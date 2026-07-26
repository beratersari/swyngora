import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { ExchangeTabs } from '@/components/organisms/ExchangeTabs';
import { MarketsToolbar } from '@/components/organisms/MarketsToolbar';
import { MarketsTable } from '@/components/organisms/MarketsTable';
import { getResultsRange } from '@/components/organisms/MarketsTable/MarketsTable.helpers';
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
  const { t } = useTranslation(['markets', 'common']);
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

  const [cachedItems, setCachedItems] = useState<SpotMarket[]>([]);
  const [cachedTotal, setCachedTotal] = useState(0);
  const filterKey = `${state.exchange}|${state.quote}|${state.tag}|${debouncedQ}|${state.sort}|${state.order}`;

  useEffect(() => {
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
        resource: t('markets:resource'),
        statusMessages: {
          502: t('markets:errors.upstreamMcap'),
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

  const range = getResultsRange(state.offset, state.limit, total);
  const rangeLabel =
    range.kind === 'empty'
      ? t('markets:results.emptyMatches')
      : t('markets:results.range', {
          from: range.from.toLocaleString(),
          to: range.to.toLocaleString(),
          total: range.total.toLocaleString(),
        });
  const isInitialLoading = spotQuery.isLoading && items.length === 0;
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
              quote: state.quote || DEFAULT_MARKETS_STATE.quote,
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
