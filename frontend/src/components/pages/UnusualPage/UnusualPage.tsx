import { Alert, Select } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { PageHeader } from '@/components/molecules/PageHeader';
import { UnusualScanBoard } from '@/components/organisms/UnusualScanBoard';
import { rtkErrorMessage, useScanMarketUnusualQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import {
  UNUSUAL_EXCHANGES,
  UNUSUAL_LIMITS,
  UNUSUAL_METRICS,
  UNUSUAL_POLL_MS,
  type UnusualExchange,
  type UnusualMetric,
} from './constants';
import { Field, PageStack, Toolbar } from './UnusualPage.styles';

export function UnusualPage() {
  const { t } = useTranslation(['unusual', 'common']);
  const visible = useDocumentVisible();
  const [exchange, setExchange] = useState<UnusualExchange>('all');
  const [metric, setMetric] = useState<UnusualMetric>('all');
  const [limit, setLimit] = useState<(typeof UNUSUAL_LIMITS)[number]>(36);

  const query = useScanMarketUnusualQuery(
    { exchange, metric, limit },
    { pollingInterval: visible ? UNUSUAL_POLL_MS : 0, refetchOnFocus: true },
  );
  const hits = query.data?.hits ?? [];
  const skipped = query.data?.skipped ?? 0;

  return (
    <PageStack>
      <PageHeader title={t('unusual:title')} subtitle={t('unusual:subtitle')} />
      <Toolbar>
        <Field>
          <Text variant="caption" color="secondary" id="unusual-exchange-label">
            {t('unusual:exchange')}
          </Text>
          <Select
            aria-labelledby="unusual-exchange-label"
            value={exchange}
            style={{ minWidth: 140 }}
            options={UNUSUAL_EXCHANGES.map((value) => ({
              value,
              label: value === 'all' ? t('unusual:allVenues') : t(`common:exchanges.${value}`),
            }))}
            onChange={(value) => setExchange(value)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="unusual-metric-label">
            {t('unusual:metric')}
          </Text>
          <Select
            aria-labelledby="unusual-metric-label"
            value={metric}
            style={{ minWidth: 160 }}
            options={UNUSUAL_METRICS.map((value) => ({
              value,
              label: value === 'all' ? t('unusual:allMetrics') : t(`unusual:metrics.${value}`),
            }))}
            onChange={(value) => setMetric(value)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary" id="unusual-limit-label">
            {t('unusual:limit')}
          </Text>
          <Select
            aria-labelledby="unusual-limit-label"
            value={limit}
            style={{ minWidth: 100 }}
            options={UNUSUAL_LIMITS.map((value) => ({ value, label: String(value) }))}
            onChange={(value) => setLimit(value)}
          />
        </Field>
      </Toolbar>

      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('unusual:loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('unusual:resource') })}
        />
      ) : null}

      {query.isSuccess ? (
        <Text variant="caption" color="secondary">
          {t('unusual:counts', {
            scanned: query.data?.scanned ?? 0,
            standouts: hits.length,
            quiet: query.data?.quiet ?? 0,
          })}
          {skipped > 0 ? ` ${t('unusual:skipped', { count: skipped })}` : ''}
        </Text>
      ) : null}

      <Text variant="caption" color="secondary">
        {t('unusual:note')}
      </Text>

      <UnusualScanBoard hits={hits} loading={query.isLoading} />
    </PageStack>
  );
}
