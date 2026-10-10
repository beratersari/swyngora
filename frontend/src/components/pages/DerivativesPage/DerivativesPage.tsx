import { Alert, Segmented, Select } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { VenueCoverage } from '@/components/molecules/VenueCoverage';
import {
  DERIVATIVES_EXCHANGES,
  DERIVATIVES_LIMITS,
  DerivativesBoard,
  type DerivativesExchange,
} from '@/components/organisms/DerivativesBoard';
import {
  DEFAULT_HEATMAP_METRIC,
  DerivativesHeatmap,
  HEATMAP_METRICS,
  type HeatMetric,
} from '@/components/organisms/DerivativesHeatmap';
import {
  DEFAULT_GRID_METRIC,
  DEFAULT_GRID_RANGE,
  DerivativesTimeGrid,
  GRID_METRICS,
  GRID_RANGES,
  type GridMetric,
  type GridRange,
} from '@/components/organisms/DerivativesTimeGrid';
import {
  rtkErrorMessage,
  useGetMarketDerivativesBoardQuery,
  useGetMarketDerivativesGridQuery,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { Field, PageStack, Toolbar } from './DerivativesPage.styles';

const POLL_MS = 20_000;

export function DerivativesPage() {
  const { t } = useTranslation(['derivatives', 'common']);
  const visible = useDocumentVisible();
  const [exchange, setExchange] = useState<DerivativesExchange>('all');
  const [limit, setLimit] = useState<(typeof DERIVATIVES_LIMITS)[number]>(36);
  const [view, setView] = useState<'table' | 'heatmap' | 'time'>('table');
  const [metric, setMetric] = useState<HeatMetric>(DEFAULT_HEATMAP_METRIC);
  const [gridMetric, setGridMetric] = useState<GridMetric>(DEFAULT_GRID_METRIC);
  const [gridRange, setGridRange] = useState<GridRange>(DEFAULT_GRID_RANGE);

  const query = useGetMarketDerivativesBoardQuery(
    { exchange, limit },
    { skip: view === 'time', pollingInterval: visible && view !== 'time' ? POLL_MS : 0, refetchOnFocus: true },
  );
  const gridQuery = useGetMarketDerivativesGridQuery(
    { exchange, metric: gridMetric, range: gridRange, limit },
    { skip: view !== 'time', pollingInterval: visible && view === 'time' ? POLL_MS : 0, refetchOnFocus: true },
  );
  const active = view === 'time' ? gridQuery : query;

  return (
    <PageStack>
      <PageHeader title={t('derivatives:title')} subtitle={t('derivatives:subtitle')} />
      <Toolbar>
        <Segmented
          value={view}
          onChange={(value) => setView(value === 'heatmap' || value === 'time' ? value : 'table')}
          options={[
            { value: 'table', label: t('derivatives:viewTable') },
            { value: 'heatmap', label: t('derivatives:viewHeatmap') },
            { value: 'time', label: t('derivatives:viewTime') },
          ]}
        />
        <Field>
          <Text variant="caption" color="secondary" id="derivatives-exchange-label">
            {t('derivatives:exchange')}
          </Text>
          <Select
            aria-labelledby="derivatives-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={DERIVATIVES_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('derivatives:allVenues') : t(`common:exchanges.${value}`),
            }))}
            onChange={(value) => setExchange(value)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="derivatives-limit-label">
            {t('derivatives:limit')}
          </Text>
          <Select
            aria-labelledby="derivatives-limit-label"
            value={limit}
            style={{ minWidth: 100 }}
            options={DERIVATIVES_LIMITS.map((value) => ({ value, label: String(value) }))}
            onChange={(value) => setLimit(value)}
          />
        </Field>
        {view === 'heatmap' ? (
          <Field>
            <Text variant="caption" color="secondary" id="derivatives-metric-label">
              {t('derivatives:metric')}
            </Text>
            <Select
              aria-labelledby="derivatives-metric-label"
              value={metric}
              virtual={false}
              listHeight={420}
              style={{ minWidth: 180 }}
              options={HEATMAP_METRICS.map((value) => ({
                value,
                label: t(`derivatives:${metricLabelKey(value)}`),
              }))}
              onChange={(value) => setMetric(value)}
            />
          </Field>
        ) : null}
        {view === 'time' ? (
          <>
            <Field>
              <Text variant="caption" color="secondary" id="derivatives-grid-metric-label">
                {t('derivatives:gridSeries')}
              </Text>
              <Select
                aria-labelledby="derivatives-grid-metric-label"
                value={gridMetric}
                style={{ minWidth: 180 }}
                options={GRID_METRICS.map((value) => ({
                  value,
                  label: t(`derivatives:${gridMetricKey(value)}`),
                }))}
                onChange={(value) => setGridMetric(value)}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary" id="derivatives-grid-range-label">
                {t('derivatives:gridRange')}
              </Text>
              <Select
                aria-labelledby="derivatives-grid-range-label"
                value={gridRange}
                style={{ minWidth: 120 }}
                options={GRID_RANGES.map((value) => ({
                  value,
                  label: t(`derivatives:${value === '7d' ? 'range7d' : 'range24h'}`),
                }))}
                onChange={(value) => setGridRange(value)}
              />
            </Field>
          </>
        ) : null}
      </Toolbar>

      {active.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('derivatives:loadFailed')}
          description={rtkErrorMessage(active.error, { resource: t('derivatives:resource') })}
        />
      ) : null}

      {view !== 'time' && query.isSuccess && query.data?.coverage?.complete === false ? (
        <VenueCoverage coverage={query.data.coverage} />
      ) : null}

      <Text variant="caption" color="secondary">
        {exchange === 'hyperliquid'
          ? t('derivatives:hyperliquidHint')
          : view === 'time'
            ? t('derivatives:gridHint')
            : view === 'heatmap'
              ? t('derivatives:heatmapHint')
              : t('derivatives:withheld')}
      </Text>

      {view === 'time' ? (
        <DerivativesTimeGrid
          exchange={exchange}
          metric={gridMetric}
          range={gridRange}
          grid={gridQuery.data}
          loading={gridQuery.isLoading}
        />
      ) : view === 'heatmap' ? (
        <DerivativesHeatmap
          exchange={exchange}
          coins={query.data?.coins ?? []}
          metric={metric}
          loading={query.isLoading}
        />
      ) : (
        <DerivativesBoard
          exchange={exchange}
          coins={query.data?.coins ?? []}
          loading={query.isLoading}
        />
      )}
    </PageStack>
  );
}

function gridMetricKey(metric: GridMetric): string {
  switch (metric) {
    case 'oi':
      return 'gridOi';
    case 'cvd':
      return 'gridCvd';
    case 'liquidations':
      return 'gridLiquidations';
    default:
      return 'gridFunding';
  }
}

function metricLabelKey(metric: HeatMetric): string {
  switch (metric) {
    case 'takerBuyPct':
      return 'takerBuy';
    case 'gapPct':
      return 'gap';
    case 'printChange':
      return 'printSize';
    case 'bidPct':
      return 'bid';
    default:
      return metric;
  }
}
