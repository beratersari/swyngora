import { useMemo, useState } from 'react';
import { Alert, Button, Select } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { VolumeSurgeTable } from '@/components/organisms/VolumeSurgeTable';
import {
  rtkErrorMessage,
  useScanMarketVolumeSurgesQuery,
  type MarketExchange,
} from '@/libs/api';
import { surgeHitsToRows } from './SurgesPage.helpers';
import { Field, PageStack, Toolbar } from './SurgesPage.styles';

export function SurgesPage() {
  const { t } = useTranslation(['surges', 'common']);
  const navigate = useNavigate();
  const [exchange, setExchange] = useState<MarketExchange>('binance');
  const [hasScanned, setHasScanned] = useState(false);

  const scan = useScanMarketVolumeSurgesQuery(
    { exchange, minRatio: 2, limit: 30 },
    { skip: !hasScanned },
  );

  const rows = useMemo(() => surgeHitsToRows(scan.data?.hits), [scan.data]);
  const hitCount = rows.length;

  return (
    <PageStack>
      <PageHeader title={t('surges:title')} subtitle={t('surges:subtitle')} />

      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="surges-exchange-label">
            {t('surges:exchange')}
          </Text>
          <Select
            aria-labelledby="surges-exchange-label"
            value={exchange}
            style={{ minWidth: 120 }}
            options={[
              { value: 'binance', label: t('common:exchanges.binance') },
              { value: 'bybit', label: t('common:exchanges.bybit') },
              { value: 'coinbase', label: t('common:exchanges.coinbase') },
            ]}
            onChange={setExchange}
          />
        </Field>
        <Button
          type="primary"
          loading={scan.isFetching}
          onClick={() => {
            if (!hasScanned) {
              setHasScanned(true);
              return;
            }
            void scan.refetch();
          }}
        >
          {t('surges:scan')}
        </Button>
      </Toolbar>

      {hasScanned && scan.isSuccess ? (
        <Text variant="caption" color="secondary">
          {t('surges:hits', { count: hitCount })} · {t('surges:note')}
        </Text>
      ) : null}

      {scan.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('surges:loadFailed')}
          description={rtkErrorMessage(scan.error, { resource: t('surges:resource') })}
        />
      ) : null}

      {scan.data?.summary ? (
        <Text variant="caption" color="secondary">
          {scan.data.summary}
        </Text>
      ) : null}

      <VolumeSurgeTable
        rows={rows}
        loading={scan.isFetching}
        hasScanned={hasScanned}
        emptyHint={t('surges:emptyHint')}
        emptyTitle={t('surges:emptyTitle')}
        columns={{
          symbol: t('surges:symbol'),
          maxRatio: t('surges:maxRatio'),
          hottest: t('surges:hottest'),
          grade: t('surges:grade'),
          dominant: t('surges:dominant'),
          summary: t('surges:read'),
        }}
        onRowOpen={(ex, symbol) =>
          navigate(`/markets/${encodeURIComponent(ex)}/${encodeURIComponent(symbol)}?tab=flow`)
        }
      />
    </PageStack>
  );
}
