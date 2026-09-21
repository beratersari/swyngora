import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  CoinList,
  CoinRow,
  Panel,
  Stat,
  StatsRow,
} from '@/components/organisms/CoinInsightPanel/CoinInsightPanel.styles';
import { windowMix } from './helpers';
import type { BreadthOverviewProps } from './BreadthOverview.types';

export function BreadthOverview({ data, error, isLoading = false }: BreadthOverviewProps) {
  const { t } = useTranslation('markets');
  const hour = data?.windows?.find((w) => w.window === '1h') ?? data?.windows?.[0];
  const extraWindows = (data?.windows ?? []).filter((w) => w.window !== hour?.window);

  return (
    <Panel>
      <Text variant="label" color="primary">
        {t('breadth.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('breadth.subtitle', { n: data?.universe ?? '—' })}
      </Text>
      {error ? (
        <Alert type="info" showIcon message={t('breadth.title')} description={error} />
      ) : (
        <>
          <StatsRow>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('breadth.hour')}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="80%">
                {windowMix(hour)}
              </Text>
            </Stat>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('breadth.align')}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="50%">
                {hour?.alignment ? t(`breadth.align_${hour.alignment}`, { defaultValue: hour.alignment }) : '—'}
              </Text>
            </Stat>
          </StatsRow>
          {extraWindows.length > 0 ? (
            <CoinList>
              {extraWindows.map((w) => (
                <CoinRow key={w.window}>
                  <Text variant="caption" color="secondary">
                    {w.window}
                  </Text>
                  <Text variant="numeric" color="primary">
                    {windowMix(w)}
                  </Text>
                </CoinRow>
              ))}
            </CoinList>
          ) : null}
          {data?.summary ? (
            <Text variant="caption" color="secondary">
              {data.summary}
            </Text>
          ) : null}
        </>
      )}
    </Panel>
  );
}
