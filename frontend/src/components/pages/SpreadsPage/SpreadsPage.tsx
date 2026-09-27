import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { SpreadsTable } from '@/components/organisms/SpreadsTable';
import { rtkErrorMessage, useScanWideSpreadsQuery } from '@/libs/api';
import { SPREAD_EXCHANGES, SPREAD_SCAN_LIMIT, type SpreadExchange } from './constants';
import { hitsToRows } from './helpers';
import { Field, PageStack, Toolbar } from './SpreadsPage.styles';

export function SpreadsPage() {
  const { t } = useTranslation(['spreads', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<SpreadExchange>('all');
  const [hasLoaded, setHasLoaded] = useState(false);

  const query = useScanWideSpreadsQuery(
    { exchange, quote: 'USDT', limit: SPREAD_SCAN_LIMIT },
    { skip: !hasLoaded, pollingInterval: hasLoaded ? 20_000 : 0 },
  );
  const rows = useMemo(() => hitsToRows(query.data?.hits), [query.data?.hits]);

  return (
    <PageStack>
      <PageHeader title={t('spreads:title')} subtitle={t('spreads:subtitle')} />
      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="spreads-exchange-label">
            {t('spreads:exchange')}
          </Text>
          <Select
            aria-labelledby="spreads-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={SPREAD_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('spreads:allVenues') : t(`common:exchanges.${value}`),
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
          {t('spreads:load')}
        </Button>
      </Toolbar>
      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('spreads:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('spreads:resource') })}
        />
      ) : null}
      {hasLoaded && query.data?.summary ? (
        <Text variant="caption" color="secondary">
          {query.data.summary}
        </Text>
      ) : null}
      <SpreadsTable
        rows={rows}
        loading={query.isFetching}
        hasLoaded={hasLoaded && !query.isLoading}
        emptyTitle={t('spreads:emptyTitle')}
        emptyHint={t('spreads:emptyHint')}
        columns={{
          symbol: t('spreads:symbol'),
          exchange: t('spreads:exchange'),
          current: t('spreads:current'),
          typical: t('spreads:typical'),
          ratio: t('spreads:ratio'),
        }}
        onRowOpen={(ex, symbol) => navigate(`/markets/${ex}/${symbol}?tab=orderbook`)}
      />
    </PageStack>
  );
}
