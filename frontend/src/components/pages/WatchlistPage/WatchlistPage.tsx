import { Alert, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
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
import { PageIntro, PageStack, ToolbarRow } from './WatchlistPage.styles';

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
  const { spot, isLoading } = useWatchlistSpot(exchange, symbol);
  return (
    <SpotMetricValue metric={metric} spot={spot} exchange={exchange} isLoading={isLoading} />
  );
}

export function WatchlistPage() {
  const { t } = useTranslation(['watchlist', 'markets', 'common']);
  const navigate = useNavigate();
  const wl = useGetWatchlistQuery();
  const [removeItem, removeState] = useRemoveWatchlistItemMutation();
  const metricColumns = useSpotMetricColumns('watchlist');

  const items = wl.data?.items ?? [];

  return (
    <PageStack>
      <PageIntro>
        <Text variant="h2" color="primary">
          {t('watchlist:title')}
        </Text>
        <Text variant="body" color="secondary">
          {t('watchlist:subtitle')}
        </Text>
      </PageIntro>

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
          void removeItem({ exchange, symbol });
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