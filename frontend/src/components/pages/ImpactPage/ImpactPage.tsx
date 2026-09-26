import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { ImpactTable } from '@/components/organisms/ImpactTable';
import { rtkErrorMessage, useScanMarketImpactQuery } from '@/libs/api';
import {
  IMPACT_EXCHANGES,
  IMPACT_LIMIT,
  IMPACT_NOTIONALS,
  IMPACT_SIDES,
  type ImpactExchange,
  type ImpactSide,
} from './constants';
import { hitsToRows } from './helpers';
import { Field, PageStack, Toolbar } from './ImpactPage.styles';

export function ImpactPage() {
  const { t } = useTranslation(['impact', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<ImpactExchange>('all');
  const [side, setSide] = useState<ImpactSide>('buy');
  const [notional, setNotional] = useState<number>(100_000);
  const [hasLoaded, setHasLoaded] = useState(false);

  const query = useScanMarketImpactQuery(
    { exchange, quote: 'USDT', side, notional, limit: IMPACT_LIMIT },
    { skip: !hasLoaded },
  );

  const rows = useMemo(
    () => hitsToRows(query.data?.hits, exchange, t('impact:yes'), t('impact:no')),
    [query.data?.hits, exchange, t],
  );

  return (
    <PageStack>
      <PageHeader title={t('impact:title')} subtitle={t('impact:subtitle')} />
      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="impact-exchange-label">
            {t('impact:exchange')}
          </Text>
          <Select
            aria-labelledby="impact-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={IMPACT_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('impact:allVenues') : t(`common:exchanges.${value}`),
            }))}
            onChange={(value) => {
              setExchange(value);
              setHasLoaded(false);
            }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="impact-side-label">
            {t('impact:side')}
          </Text>
          <Select
            aria-labelledby="impact-side-label"
            value={side}
            style={{ minWidth: 120 }}
            options={IMPACT_SIDES.map((value) => ({
              value,
              label: t(value === 'buy' ? 'impact:buy' : 'impact:sell'),
            }))}
            onChange={(value) => {
              setSide(value);
              setHasLoaded(false);
            }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="impact-size-label">
            {t('impact:size')}
          </Text>
          <Select
            aria-labelledby="impact-size-label"
            value={notional}
            style={{ minWidth: 140 }}
            options={IMPACT_NOTIONALS.map((value) => ({
              value,
              label: t('impact:notional', { size: value.toLocaleString('en-US') }),
            }))}
            onChange={(value) => {
              setNotional(value);
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
          {t('impact:load')}
        </Button>
      </Toolbar>

      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('impact:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('impact:resource') })}
        />
      ) : null}

      {hasLoaded && query.data?.summary ? (
        <Text variant="caption" color="secondary">
          {query.data.summary}
        </Text>
      ) : null}

      <ImpactTable
        rows={rows}
        loading={query.isFetching}
        hasLoaded={hasLoaded}
        emptyHint={t('impact:emptyHint')}
        emptyTitle={t('impact:emptyTitle')}
        columns={{
          symbol: t('impact:symbol'),
          slippage: t('impact:slippage'),
          versusBest: t('impact:versusBest'),
          impact: t('impact:touch'),
          exhausted: t('impact:exhausted'),
        }}
        onRowOpen={(ex, symbol) =>
          navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(symbol)}?tab=orderbook`)
        }
      />
    </PageStack>
  );
}
