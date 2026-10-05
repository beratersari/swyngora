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
import { rtkErrorMessage, useGetMarketDerivativesBoardQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { Field, PageStack, Toolbar } from './DerivativesPage.styles';

const POLL_MS = 20_000;

export function DerivativesPage() {
  const { t } = useTranslation(['derivatives', 'common']);
  const visible = useDocumentVisible();
  const [exchange, setExchange] = useState<DerivativesExchange>('all');
  const [limit, setLimit] = useState<(typeof DERIVATIVES_LIMITS)[number]>(36);
  const [view, setView] = useState<'table' | 'heatmap'>('table');
  const [metric, setMetric] = useState<HeatMetric>(DEFAULT_HEATMAP_METRIC);

  const query = useGetMarketDerivativesBoardQuery(
    { exchange, limit },
    { pollingInterval: visible ? POLL_MS : 0, refetchOnFocus: true },
  );

  return (
    <PageStack>
      <PageHeader title={t('derivatives:title')} subtitle={t('derivatives:subtitle')} />
      <Toolbar>
        <Segmented
          value={view}
          onChange={(value) => setView(value === 'heatmap' ? 'heatmap' : 'table')}
          options={[
            { value: 'table', label: t('derivatives:viewTable') },
            { value: 'heatmap', label: t('derivatives:viewHeatmap') },
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
      </Toolbar>

      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('derivatives:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('derivatives:resource') })}
        />
      ) : null}

      {query.isSuccess && query.data?.coverage?.complete === false ? (
        <VenueCoverage coverage={query.data.coverage} />
      ) : null}

      <Text variant="caption" color="secondary">
        {view === 'heatmap' ? t('derivatives:heatmapHint') : t('derivatives:withheld')}
      </Text>

      {view === 'heatmap' ? (
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
