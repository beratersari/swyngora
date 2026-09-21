import { Alert } from 'antd';
import { Text } from '@/components/atoms/Text';
import { Panel, Stat, StatsRow } from './CoinInsightPanel.styles';
import type { CoinInsightPanelProps } from './CoinInsightPanel.types';

export function CoinInsightPanel({
  title,
  summary,
  stats,
  error,
  isLoading = false,
}: CoinInsightPanelProps) {
  return (
    <Panel>
      <Text variant="label" color="primary">
        {title}
      </Text>
      {error ? (
        <Alert type="info" showIcon message={title} description={error} />
      ) : (
        <>
          {stats && stats.length > 0 ? (
            <StatsRow>
              {stats.map((s) => (
                <Stat key={s.label}>
                  <Text variant="caption" color="secondary">
                    {s.label}
                  </Text>
                  <Text variant="numeric" color="primary" isLoading={isLoading} skeletonWidth="70%">
                    {s.value}
                  </Text>
                </Stat>
              ))}
            </StatsRow>
          ) : null}
          {summary ? (
            <Text variant="caption" color="secondary">
              {summary}
            </Text>
          ) : null}
        </>
      )}
    </Panel>
  );
}
