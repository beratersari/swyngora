import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatCompactAmount } from '@/libs/utils';
import { leaderCoins, shareLabel } from './helpers';
import { CoinList, CoinRow, Panel, Stat, StatsRow } from './PerpLiquidityOverview.styles';
import type { PerpLiquidityOverviewProps } from './PerpLiquidityOverview.types';

export function PerpLiquidityOverview({ data, error, isLoading = false }: PerpLiquidityOverviewProps) {
  const { t } = useTranslation('markets');
  const leaders = leaderCoins(data);
  const leanKey = data?.lean === 'bid' ? 'perpLiq.leanBid' : data?.lean === 'ask' ? 'perpLiq.leanAsk' : 'perpLiq.leanEven';

  return (
    <Panel>
      <Text variant="label" color="primary">
        {t('perpLiq.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('perpLiq.subtitle', { pct: data?.rangePct ?? '1' })}
      </Text>
      {error ? (
        <Alert type="info" showIcon message={t('perpLiq.title')} description={error} />
      ) : (
        <>
          <StatsRow>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('perpLiq.bid')}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
                {formatCompactAmount(data?.bid)}
              </Text>
            </Stat>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('perpLiq.ask')}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
                {formatCompactAmount(data?.ask)}
              </Text>
            </Stat>
            <Stat>
              <Text variant="caption" color="secondary">
                {t('perpLiq.lean')}
              </Text>
              <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="50%">
                {data?.lean ? t(leanKey) : '—'}
              </Text>
            </Stat>
          </StatsRow>
          {leaders.length > 0 ? (
            <CoinList>
              {leaders.map((c) => (
                <CoinRow key={c.symbol}>
                  <Text variant="caption" color="secondary">
                    {c.base || c.symbol}
                  </Text>
                  <Text variant="numeric" color="primary">
                    {shareLabel(c.sharePct)}
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
