import { Alert, Button, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { MetricColumnPicker } from '@/components/molecules/MetricColumnPicker';
import { SpotMetricValue } from '@/components/molecules/SpotMetricValue';
import { WatchlistTable } from '@/components/organisms/WatchlistTable';
import {
  rtkErrorMessage,
  useGetWatchlistQuery,
  useRemoveWatchlistItemMutation,
} from '@/libs/api';
import { useSpotMetricColumns, useWatchlistSpot } from '@/libs/hooks';
import { metricColumnTitle, type SpotMetricDef } from '@/libs/utils';
import { PageStack, ToolbarRow } from './WatchlistPage.styles';

/** Page-owned live metric cell — RTK stays out of organisms. */
function WatchlistLiveMetric({
  exchange,
  symbol,
  metric,
}: {
  exchange: string;
  symbol: string;
  metric: SpotMetricDef;
}) {
  const { t } = useTranslation('watchlist');
  const { spot, isLoading, isError } = useWatchlistSpot(exchange, symbol);
  if (isError && !spot) {
    return (
      <Text variant="caption" color="secondary" title={t('metricFailed')}>
        —
      </Text>
    );
  }
  return (
    <SpotMetricValue metric={metric} spot={spot} exchange={exchange} isLoading={isLoading} />
  );
}

export function WatchlistPage() {
  const { t } = useTranslation(['watchlist', 'markets', 'common']);
  const navigate = useNavigate();
  const wl = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const [removeItem, removeState] = useRemoveWatchlistItemMutation();
  const metricColumns = useSpotMetricColumns('watchlist');

  const items = wl.data?.items ?? [];

  return (
    <PageStack>
      <PageHeader title={t('watchlist:title')} />

      {wl.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('watchlist:loadFailed')}
          description={rtkErrorMessage(wl.error, { resource: t('watchlist:resource') })}
          action={
            <Button size="small" onClick={() => void wl.refetch()}>
              {t('common:actions.retry')}
            </Button>
          }
        />
      ) : null}

      {removeState.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('watchlist:removeFailed')}
          description={rtkErrorMessage(removeState.error, {
            resource: t('watchlist:resource'),
          })}
        />
      ) : null}

      <ToolbarRow>
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
      </ToolbarRow>

      <WatchlistTable
        items={items}
        loading={wl.isLoading}
        removeLoading={removeState.isLoading}
        metrics={metricColumns.metrics}
        renderMetric={({ exchange, symbol, metric }) => (
          <WatchlistLiveMetric exchange={exchange} symbol={symbol} metric={metric} />
        )}
        onRemove={(exchange, symbol) => {
          void removeItem({ exchange, symbol })
            .unwrap()
            .catch((err) => {
              void message.error(
                rtkErrorMessage(err, { resource: t('watchlist:resource') }),
              );
            });
        }}
        onOpen={(exchange, symbol) => {
          navigate(
            `/markets/${encodeURIComponent(exchange)}/${encodeURIComponent(symbol)}`,
          );
        }}
      />
    </PageStack>
  );
}