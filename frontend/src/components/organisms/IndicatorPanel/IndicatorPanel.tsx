import { Alert, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { IndicatorChartHost } from '@/components/molecules/IndicatorChartHost';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  formatIndicator,
  indicatorPointsToRsiLine,
  rsiBandKey,
  rsiTone,
  sortedEmaKeys,
} from '@/libs/utils';
import { emaColor } from './IndicatorPanel.helpers';
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

export function IndicatorPanel({
  data,
  priceQuote,
  errorMessage,
  isLoading = false,
  showEmaOnChart = true,
  onToggleEma,
}: IndicatorPanelProps) {
  const { t } = useTranslation('detail');
  const { formatPrice } = useDisplayCurrency();
  const rsi = data?.latest?.rsi;
  const zone = data?.latest?.zone;
  const emaKeys = sortedEmaKeys(data?.latest?.ema);
  const rsiLine = indicatorPointsToRsiLine(data?.points);
  const period = data?.rsiPeriod ?? 14;
  // Empty when no EMA keys yet — avoid hard-coded periods that may not exist.
  const emaPeriodLabel = emaKeys.join(', ') || '—';
  const band = t(`indicators.band.${rsiBandKey(zone)}`);

  if (errorMessage) {
    return (
      <Panel>
        <Alert
          type="error"
          showIcon
          message={t('indicators.unavailable')}
          description={errorMessage}
        />
      </Panel>
    );
  }

  return (
    <Panel>
      <PanelHead>
        <div>
          <Text variant="h4" color="primary">
            {t('indicators.title')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('indicators.subtitle', {
              rsiPeriod: period,
              emaPeriods: emaPeriodLabel,
            })}
          </Text>
        </div>
        {onToggleEma ? (
          <label style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch size="small" checked={showEmaOnChart} onChange={onToggleEma} />
            <Text variant="caption" color="secondary">
              {t('indicators.emaOnChart')}
            </Text>
          </label>
        ) : null}
      </PanelHead>

      <SnapshotGrid>
        <SnapshotCard>
          <Text variant="caption" color="secondary">
            {t('indicators.rsiLatest', { period })}
          </Text>
          <Text
            variant="h3"
            color={rsiTone(zone)}
            mono
            isLoading={isLoading}
            skeletonWidth={80}
          >
            {formatIndicator(rsi)}
          </Text>
          <Text variant="caption" color="secondary">
            {t('indicators.rsiBand', { band })}
          </Text>
        </SnapshotCard>
        {emaKeys.map((key, i) => (
          <SnapshotCard key={key}>
            <Text variant="caption" color="secondary">
              <LegendSwatch $color={emaColor(key, i)} />
              {t('indicators.emaLatest', { period: key })}
            </Text>
            <Text variant="h3" color="primary" mono isLoading={isLoading} skeletonWidth={90}>
              {priceQuote
                ? formatPrice(data?.latest?.ema?.[key], priceQuote)
                : formatIndicator(data?.latest?.ema?.[key], 4)}
            </Text>
          </SnapshotCard>
        ))}
      </SnapshotGrid>

      <ChartBlock>
        <Text variant="label" color="secondary">
          {t('indicators.rsiSeries')}
        </Text>
        {rsiLine.length === 0 && !isLoading ? (
          <Text variant="caption" color="secondary">
            {t('indicators.rsiEmpty')}
          </Text>
        ) : (
          <IndicatorChartHost data={rsiLine} isLoading={isLoading && rsiLine.length === 0} />
        )}
      </ChartBlock>

      {showEmaOnChart && emaKeys.length > 0 ? (
        <LegendRow>
          <Text variant="caption" color="secondary">
            {t('indicators.emaLegend')}
          </Text>
          {emaKeys.map((key, i) => (
            <Text key={key} variant="caption" color="primary">
              <LegendSwatch $color={emaColor(key, i)} />
              {t('indicators.emaLabel', { period: key })}
            </Text>
          ))}
        </LegendRow>
      ) : null}

      <Text variant="caption" color="secondary">
        {data?.note?.trim() || t('indicators.fallbackNote')}
      </Text>
    </Panel>
  );
}

export { emaColor } from './IndicatorPanel.helpers';
