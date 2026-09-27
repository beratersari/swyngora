import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { GapsTable } from '@/components/organisms/GapsTable';
import { rtkErrorMessage, useScanBookGapsQuery } from '@/libs/api';
import { GAP_EXCHANGES, GAP_SCAN_LIMIT, type GapExchange } from './constants';
import { hitsToRows } from './helpers';
import { Field, PageStack, Toolbar } from './GapsPage.styles';

export function GapsPage() {
  const { t } = useTranslation(['gaps', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<GapExchange>('all');
  const [hasLoaded, setHasLoaded] = useState(false);

  const query = useScanBookGapsQuery(
    { exchange, quote: 'USDT', limit: GAP_SCAN_LIMIT, rangePct: 2 },
    { skip: !hasLoaded, pollingInterval: hasLoaded ? 20_000 : 0 },
  );

  const rows = useMemo(
    () =>
      hitsToRows(query.data?.hits, exchange, (easier) =>
        t(easier === 'up' ? 'gaps:up' : easier === 'down' ? 'gaps:down' : 'gaps:even'),
      ),
    [query.data?.hits, exchange, t],
  );

  return (
    <PageStack>
      <PageHeader title={t('gaps:title')} subtitle={t('gaps:subtitle')} />
      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="gaps-exchange-label">
            {t('gaps:exchange')}
          </Text>
          <Select
            aria-labelledby="gaps-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={GAP_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('gaps:allVenues') : t(`common:exchanges.${value}`),
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
          {t('gaps:load')}
        </Button>
      </Toolbar>
      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('gaps:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('gaps:resource') })}
        />
      ) : null}
      {hasLoaded && query.data?.summary ? (
        <Text variant="caption" color="secondary">
          {query.data.summary}
        </Text>
      ) : null}
      <GapsTable
        rows={rows}
        loading={query.isFetching}
        hasLoaded={hasLoaded}
        emptyHint={t('gaps:emptyHint')}
        emptyTitle={t('gaps:emptyTitle')}
        columns={{
          symbol: t('gaps:symbol'),
          easier: t('gaps:easier'),
          above: t('gaps:above'),
          below: t('gaps:below'),
          emptyAbove: t('gaps:emptyAbove'),
          emptyBelow: t('gaps:emptyBelow'),
          widest: t('gaps:widest'),
        }}
        onRowOpen={(ex, symbol) =>
          navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(symbol)}?tab=orderbook`)
        }
      />
    </PageStack>
  );
}
