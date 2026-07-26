import { Alert, Switch } from 'antd';
import { Text } from '@/components/atoms/Text';
import { IndicatorChartHost } from '@/components/molecules/IndicatorChartHost';
import {
  formatIndicator,
  indicatorPointsToRsiLine,
  rsiBandLabel,
  rsiTone,
  sortedEmaKeys,
} from '@/libs/utils';
import { EMA_COLORS, FALLBACK_EMA_COLORS } from './IndicatorPanel.constants';
import {
  ChartBlock,
  LegendRow,
  LegendSwatch,
  Panel,
  PanelHead,
  SnapshotCard,
  SnapshotGrid,
} from './IndicatorPanel.styles';
import type { IndicatorPanelProps } from './IndicatorPanel.types';

function emaColor(key: string, index: number): string {
  return EMA_COLORS[key] ?? FALLBACK_EMA_COLORS[index % FALLBACK_EMA_COLORS.length]!;
}

export function IndicatorPanel({
  data,
  errorMessage,
  isLoading = false,
  showEmaOnChart = true,
  onToggleEma,
}: IndicatorPanelProps) {
  const rsi = data?.latest?.rsi;
  const emaKeys = sortedEmaKeys(data?.latest?.ema);
  const rsiLine = indicatorPointsToRsiLine(data?.points);
  const period = data?.rsiPeriod ?? 14;

  if (errorMessage) {
    return (
      <Panel>
        <Alert type="error" showIcon message="Indicators unavailable" description={errorMessage} />
      </Panel>
    );
  }

  return (
    <Panel>
      <PanelHead>
        <div>
          <Text variant="h4" color="cream">
            Technical indicators
          </Text>
          <Text variant="caption" color="steel">
            RSI({period}) · EMA({emaKeys.join(', ') || '12, 26'}) · same interval as chart ·
            informational only — not financial advice
          </Text>
        </div>
        {onToggleEma ? (
          <label style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch size="small" checked={showEmaOnChart} onChange={onToggleEma} />
            <Text variant="caption" color="steel">
              EMA on price chart
            </Text>
          </label>
        ) : null}
      </PanelHead>

      <SnapshotGrid>
        <SnapshotCard>
          <Text variant="caption" color="steel">
            RSI({period}) latest
          </Text>
          <Text
            variant="h3"
            color={rsiTone(rsi)}
            mono
            isLoading={isLoading}
            skeletonWidth={80}
          >
            {formatIndicator(rsi)}
          </Text>
          <Text variant="caption" color="secondary">
            {rsiBandLabel(rsi)} (30 / 70 bands)
          </Text>
        </SnapshotCard>
        {emaKeys.map((key, i) => (
          <SnapshotCard key={key}>
            <Text variant="caption" color="steel">
              <LegendSwatch $color={emaColor(key, i)} />
              EMA({key}) latest
            </Text>
            <Text variant="h3" color="cream" mono isLoading={isLoading} skeletonWidth={90}>
              {formatIndicator(data?.latest?.ema?.[key], 4)}
            </Text>
          </SnapshotCard>
        ))}
      </SnapshotGrid>

      <ChartBlock>
        <Text variant="label" color="steel">
          RSI series
        </Text>
        <IndicatorChartHost data={rsiLine} isLoading={isLoading && rsiLine.length === 0} />
      </ChartBlock>

      {showEmaOnChart && emaKeys.length > 0 ? (
        <LegendRow>
          <Text variant="caption" color="steel">
            EMA legend (price chart):
          </Text>
          {emaKeys.map((key, i) => (
            <Text key={key} variant="caption" color="cream">
              <LegendSwatch $color={emaColor(key, i)} />
              EMA {key}
            </Text>
          ))}
        </LegendRow>
      ) : null}

      <Text variant="caption" color="secondary">
        {data?.note?.trim() ||
          'Informational analysis only — not financial advice. RSI uses Wilder smoothing; EMA seeded with SMA.'}
      </Text>
    </Panel>
  );
}

export { emaColor };
