import { Spin } from 'antd';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { DerivativesHeatmapProps } from './DerivativesHeatmap.types';
import { Cell, Grid, Value } from './DerivativesHeatmap.styles';
import { heatFill, heatmapCells } from './helpers';

export function DerivativesHeatmap({ exchange, coins, metric, loading }: DerivativesHeatmapProps) {
  const { t } = useTranslation('derivatives');
  const cells = useMemo(() => heatmapCells(exchange, coins, metric), [exchange, coins, metric]);
  const metricLabel = t(`derivatives:${metricLabelKey(metric)}`);

  if (!loading && cells.length === 0) {
    return <Text color="secondary">{t('derivatives:empty')}</Text>;
  }

  return (
    <Spin spinning={loading}>
      <Grid aria-label={metricLabel} data-testid="derivatives-heatmap">
        {cells.map((cell) => (
          <Cell
            key={cell.symbol}
            to={cell.path}
            data-testid="derivatives-heat-cell"
            data-tone={cell.tone}
            data-strength={cell.strength.toFixed(2)}
            title={cell.tone === 'blank' ? t('derivatives:heatmapBlank') : cell.text}
            style={{ backgroundColor: heatFill(cell.tone, cell.strength) }}
          >
            <Text variant="caption">{cell.label}</Text>
            <Value>{cell.text}</Value>
          </Cell>
        ))}
      </Grid>
    </Spin>
  );
}

function metricLabelKey(metric: DerivativesHeatmapProps['metric']): string {
  switch (metric) {
    case 'longPct':
      return 'longPct';
    case 'takerBuyPct':
      return 'takerBuy';
    case 'gapPct':
      return 'gap';
    case 'printChange':
      return 'printSize';
    case 'liqHeat':
      return 'liqHeat';
    case 'bidPct':
      return 'bid';
    default:
      return metric;
  }
}
