import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils';
import { formatChangePct } from './helpers';
import { Panel, StatCard, StatsGrid, TitleRow } from './PostDelistPanel.styles';
import type { PostDelistPanelProps } from './PostDelistPanel.types';

function Stat({
  label,
  value,
  isLoading,
}: {
  label: string;
  value: string;
  isLoading?: boolean;
}) {
  return (
    <StatCard>
      <Text variant="caption" color="secondary">
        {label}
      </Text>
      <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
        {value}
      </Text>
    </StatCard>
  );
}

export function PostDelistPanel({
  view,
  error,
  lastPrice,
  isLoading = false,
}: PostDelistPanelProps) {
  const { t } = useTranslation('detail');
  const source = view?.sourceLabel || view?.source || '—';
  const asOf = view?.asOf
    ? t('postDelist.asOf', { date: new Date(view.asOf).toISOString().slice(0, 16) + 'Z' })
    : null;

  return (
    <Panel data-testid="post-delist-panel">
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('postDelist.title')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('postDelist.subtitle')}
        </Text>
      </TitleRow>

      {error ? (
        <Alert type="warning" showIcon message={t('postDelist.emptyTitle')} description={error} />
      ) : null}

      {!error && view && !view.available && !isLoading ? (
        <Alert
          type="info"
          showIcon
          message={t('postDelist.emptyTitle')}
          description={view.note || t('postDelist.emptyBody')}
        />
      ) : null}

      {!error && (isLoading || view?.available) ? (
        <>
          {view?.note ? (
            <Alert type="info" showIcon message={source} description={view.note} />
          ) : null}
          <StatsGrid>
            <Stat
              label={t('postDelist.last')}
              value={formatPrice(lastPrice ?? view?.lastPrice)}
              isLoading={isLoading}
            />
            <Stat
              label={t('postDelist.change24h')}
              value={formatChangePct(view?.priceChangePercent)}
              isLoading={isLoading}
            />
            <Stat label={t('postDelist.source')} value={source} isLoading={isLoading} />
            <Stat
              label={t('postDelist.quote')}
              value={view?.quote ?? '—'}
              isLoading={isLoading}
            />
          </StatsGrid>
          {asOf ? (
            <Text variant="caption" color="secondary">
              {asOf}
            </Text>
          ) : null}
        </>
      ) : null}
    </Panel>
  );
}
