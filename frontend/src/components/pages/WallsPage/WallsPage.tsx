import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { WallsTable } from '@/components/organisms/WallsTable';
import { rtkErrorMessage, useScanSuspiciousWallsQuery } from '@/libs/api';
import { WALL_EXCHANGES, WALL_SCAN_LIMIT, type WallExchange } from './constants';
import { hitsToRows } from './helpers';
import { Field, PageStack, Toolbar } from './WallsPage.styles';

export function WallsPage() {
  const { t } = useTranslation(['walls', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<WallExchange>('all');
  const [hasLoaded, setHasLoaded] = useState(false);

  const query = useScanSuspiciousWallsQuery(
    { exchange, quote: 'USDT', limit: WALL_SCAN_LIMIT },
    { skip: !hasLoaded, pollingInterval: hasLoaded ? 15_000 : 0 },
  );

  const rows = useMemo(
    () =>
      hitsToRows(
        query.data?.hits,
        (kind) => t(kind === 'suspicious' ? 'walls:suspicious' : 'walls:flicker'),
        (side) => t(side === 'bid' ? 'walls:bid' : 'walls:ask'),
        t('walls:yes'),
        t('walls:no'),
      ),
    [query.data?.hits, t],
  );

  return (
    <PageStack>
      <PageHeader title={t('walls:title')} subtitle={t('walls:subtitle')} />
      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="walls-exchange-label">
            {t('walls:exchange')}
          </Text>
          <Select
            aria-labelledby="walls-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={WALL_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('walls:allVenues') : t(`common:exchanges.${value}`),
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
          {t('walls:load')}
        </Button>
      </Toolbar>

      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('walls:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('walls:resource') })}
        />
      ) : null}

      {hasLoaded && query.data?.summary ? (
        <Text variant="caption" color="secondary">
          {query.data.summary}
        </Text>
      ) : null}

      <WallsTable
        rows={rows}
        loading={query.isFetching}
        hasLoaded={hasLoaded}
        emptyHint={t('walls:emptyHint')}
        emptyTitle={t('walls:emptyTitle')}
        columns={{
          symbol: t('walls:symbol'),
          venue: t('walls:venue'),
          side: t('walls:side'),
          price: t('walls:price'),
          notional: t('walls:notional'),
          distance: t('walls:distance'),
          kind: t('walls:kind'),
          appears: t('walls:appears'),
          present: t('walls:present'),
        }}
        onRowOpen={(ex, symbol) =>
          navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(symbol)}?tab=orderbook`)
        }
      />
    </PageStack>
  );
}
