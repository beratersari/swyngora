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
import { fearGreedTone, fearGreedWindowLine, recentFearGreed } from './helpers';
import { HistoryBar, HistoryRow } from './FearGreedPanel.styles';
import type { FearGreedPanelProps } from './FearGreedPanel.types';

function barTone(level?: string): 'up' | 'down' | 'flat' {
  const tone = fearGreedTone(level);
  if (tone === 'success') return 'up';
  if (tone === 'error') return 'down';
  return 'flat';
}

export function FearGreedPanel({ data, error, isLoading = false }: FearGreedPanelProps) {
  const { t } = useTranslation('markets');
  const tone = fearGreedTone(data?.level);
  const levelLabel = data?.level
    ? t(`fearGreed.level_${data.level}`, { defaultValue: data.label ?? data.level })
    : '—';
  const top = data?.top;
  const toward = top?.toward
    ? t(`fearGreed.toward_${top.toward}`, { defaultValue: top.toward })
    : '';
  const topName = top?.id
    ? t(`fearGreed.factor_${top.id}`, { defaultValue: top.name ?? top.id })
    : '';
  const bars = recentFearGreed(data?.history);
  const windows = data?.windows ?? [];

  return (
    <Panel>
      <Text variant="label" color="primary">
        {t('fearGreed.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('fearGreed.subtitle')}
      </Text>
      {error ? (
        <Alert type="info" showIcon message={t('fearGreed.title')} description={error} />
      ) : (
        <>
          <StatsRow>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('fearGreed.score')}
              </Text>
              <Text variant="h3" color={tone} isLoading={isLoading} skeletonWidth={64}>
                {data?.score ?? '—'}
              </Text>
            </Stat>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('fearGreed.level')}
              </Text>
              <Text variant="numeric" color={tone} isLoading={isLoading} skeletonWidth="70%">
                {levelLabel}
              </Text>
            </Stat>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('fearGreed.inputs', { used: data?.used ?? '—', total: data?.total ?? 8 })}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="90%">
                {top
                  ? t('fearGreed.moved', { name: topName, pull: top.pull ?? '', toward })
                  : t('fearGreed.even')}
              </Text>
            </Stat>
          </StatsRow>
          {(data?.factors?.length ?? 0) > 0 ? (
            <CoinList>
              {data?.factors?.map((factor) => (
                <CoinRow key={factor.id ?? factor.name}>
                  <Text variant="caption" color="secondary">
                    {t(`fearGreed.factor_${factor.id ?? ''}`, { defaultValue: factor.name ?? factor.id })}
                  </Text>
                  <Text variant="numeric" color={fearGreedTone(factor.toward === 'greed' ? 'greed' : factor.toward === 'fear' ? 'fear' : 'neutral')}>
                    {factor.pull ?? '—'}
                  </Text>
                  {factor.detail ? (
                    <Text variant="caption" color="secondary">
                      {factor.detail}
                    </Text>
                  ) : null}
                </CoinRow>
              ))}
            </CoinList>
          ) : null}
          {windows.length > 0 ? (
            <CoinList>
              {windows.map((w) => {
                const line = fearGreedWindowLine(w);
                return (
                  <CoinRow key={w.window}>
                    <Text variant="caption" color="secondary">
                      {w.window}
                    </Text>
                    <Text variant="numeric" color="primary">
                      {line ?? t('fearGreed.windowWait')}
                    </Text>
                  </CoinRow>
                );
              })}
            </CoinList>
          ) : null}
          {bars.length > 0 ? (
            <>
              <Text variant="caption" color="secondary">
                {t('fearGreed.history')}
              </Text>
              <HistoryRow aria-hidden>
                {bars.map((point, i) => (
                  <HistoryBar
                    key={`${point.time ?? i}`}
                    $height={Math.round(((point.score ?? 0) / 100) * 36)}
                    $tone={barTone(point.level)}
                    title={`${point.score ?? ''}`}
                  />
                ))}
              </HistoryRow>
            </>
          ) : (
            <Text variant="caption" color="secondary">
              {t('fearGreed.historyEmpty')}
            </Text>
          )}
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
