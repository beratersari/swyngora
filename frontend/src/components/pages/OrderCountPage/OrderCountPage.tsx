import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { VenueCoverage } from '@/components/molecules/VenueCoverage';
import { OrderCountTable } from '@/components/organisms/OrderCountTable';
import { rtkErrorMessage, useGetMarketOrderCountOverviewQuery } from '@/libs/api';
import {
  ORDER_COUNT_EXCHANGES,
  ORDER_COUNT_LIMIT,
  ORDER_COUNT_WINDOWS,
  type OrderCountExchange,
  type OrderCountWindowId,
} from './constants';
import { coinsToRows, windowById } from './helpers';
import { Field, PageStack, Toolbar, WindowCard, WindowGrid } from './OrderCountPage.styles';

export function OrderCountPage() {
  const { t } = useTranslation(['ordercount', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<OrderCountExchange>('all');
  const [windowId, setWindowId] = useState<OrderCountWindowId>('5m');
  const [hasLoaded, setHasLoaded] = useState(false);

  const query = useGetMarketOrderCountOverviewQuery(
    { exchange, quote: 'USDT', limit: ORDER_COUNT_LIMIT },
    { skip: !hasLoaded },
  );

  const leanLabel = (lean: string) => {
    if (lean === 'buy') return t('ordercount:leanBuy');
    if (lean === 'sell') return t('ordercount:leanSell');
    if (lean === 'balanced') return t('ordercount:leanBalanced');
    return '—';
  };

  const rows = useMemo(
    () => coinsToRows(query.data?.coins, windowId, leanLabel),
    [query.data?.coins, windowId, t],
  );
  const active = windowById(query.data?.windows, windowId);

  return (
    <PageStack>
      <PageHeader title={t('ordercount:title')} subtitle={t('ordercount:subtitle')} />

      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="ordercount-exchange-label">
            {t('ordercount:exchange')}
          </Text>
          <Select
            aria-labelledby="ordercount-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={ORDER_COUNT_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('ordercount:allVenues') : t(`common:exchanges.${value}`),
            }))}
            onChange={(value) => {
              setExchange(value);
              setHasLoaded(false);
            }}
          />
        </Field>
        <Button
          type="primary"
          loading={query.isFetching}
          onClick={() => {
            if (!hasLoaded) {
              setHasLoaded(true);
              return;
            }
            void query.refetch();
          }}
        >
          {t('ordercount:load')}
        </Button>
      </Toolbar>

      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('ordercount:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('ordercount:resource') })}
        />
      ) : null}

      {hasLoaded && query.isSuccess && query.data?.definite === false ? (
        <VenueCoverage coverage={query.data.coverage} />
      ) : null}

      {hasLoaded && query.isSuccess && query.data?.definite !== false ? (
        <WindowGrid>
          {ORDER_COUNT_WINDOWS.map((id) => {
            const win = windowById(query.data?.windows, id);
            return (
              <WindowCard
                key={id}
                type="button"
                $active={id === windowId}
                onClick={() => setWindowId(id)}
              >
                <Text variant="caption" color="secondary">
                  {id}
                </Text>
                <Text variant="label" color="primary">
                  {win?.ratio || '—'}
                </Text>
                <Text variant="caption" color="secondary">
                  {t('ordercount:coinSplit', {
                    buy: win?.buyCoins ?? 0,
                    sell: win?.sellCoins ?? 0,
                    balanced: win?.balancedCoins ?? 0,
                    quiet: win?.quietCoins ?? 0,
                  })}
                </Text>
              </WindowCard>
            );
          })}
        </WindowGrid>
      ) : null}

      {query.data?.definite !== false && active?.summary ? (
        <Text variant="caption" color="secondary">
          {active.summary}
        </Text>
      ) : null}

      <OrderCountTable
        rows={query.data?.definite === false ? [] : rows}
        loading={query.isFetching}
        hasLoaded={hasLoaded && query.data?.definite !== false}
        emptyHint={t('ordercount:emptyHint')}
        emptyTitle={t('ordercount:emptyTitle')}
        columns={{
          symbol: t('ordercount:symbol'),
          ratio: t('ordercount:ratio'),
          lean: t('ordercount:lean'),
          buyCount: t('ordercount:buyCount'),
          sellCount: t('ordercount:sellCount'),
          summary: t('ordercount:read'),
        }}
        onRowOpen={(ex, symbol) =>
          navigate(
            `/markets/${encodeURIComponent(ex === 'all' ? 'binance' : ex)}/${encodeURIComponent(symbol)}?tab=tape`,
          )
        }
      />
    </PageStack>
  );
}
