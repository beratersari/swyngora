import { Button, Select, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { IntervalRail } from '@/components/molecules/IntervalRail';
import { DETAIL_PUMP_THRESHOLD_OPTIONS } from '@/config/constants';
import { Field, InlineField, ToolbarRow } from './DetailChartToolbar.styles';
import type { DetailChartToolbarProps } from './DetailChartToolbar.types';

export function DetailChartToolbar({
  intervals,
  interval,
  intervalsLoading,
  onIntervalChange,
  onRefresh,
  isFetching,
  pumpThresholdPct,
  onPumpThresholdChange,
  showPumpMarkers = true,
  onShowPumpMarkersChange,
  showSignalMarkers = true,
  onShowSignalMarkersChange,
}: DetailChartToolbarProps) {
  const { t } = useTranslation(['detail', 'common']);

  // Keep current value in the list even if it was set outside presets (defensive).
  const thresholdOptions = Array.from(
    new Set([...DETAIL_PUMP_THRESHOLD_OPTIONS, pumpThresholdPct].filter((n): n is number => n != null)),
  )
    .sort((a, b) => a - b)
    .map((n) => ({ value: n, label: `±${n}%` }));

  return (
    <ToolbarRow>
      <Field>
        <Text variant="caption" color="secondary">
          {t('detail:chart.interval')}
        </Text>
        <IntervalRail
          intervals={intervals}
          value={interval}
          onChange={onIntervalChange}
          loading={intervalsLoading}
          aria-label={t('detail:chart.interval')}
        />
      </Field>

      {onPumpThresholdChange != null && pumpThresholdPct != null ? (
        <Field $compact>
          <Text variant="caption" color="secondary">
            {t('detail:chart.pumpThreshold')}
          </Text>
          <Select
            size="small"
            value={pumpThresholdPct}
            options={thresholdOptions}
            onChange={(v) => onPumpThresholdChange(Number(v))}
            style={{ minWidth: 88 }}
            aria-label={t('detail:chart.pumpThreshold')}
          />
        </Field>
      ) : null}

      {onShowPumpMarkersChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.pumpMarkers')}
          </Text>
          <Switch
            size="small"
            checked={showPumpMarkers}
            onChange={onShowPumpMarkersChange}
            aria-label={t('detail:chart.pumpMarkers')}
          />
        </InlineField>
      ) : null}

      {onShowSignalMarkersChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.signalMarkers')}
          </Text>
          <Switch
            size="small"
            checked={showSignalMarkers}
            onChange={onShowSignalMarkersChange}
            aria-label={t('detail:chart.signalMarkers')}
          />
        </InlineField>
      ) : null}

      {onRefresh ? (
        <Button size="small" onClick={onRefresh} loading={isFetching}>
          {t('common:actions.refresh')}
        </Button>
      ) : null}
      {isFetching ? <BrandTag variant="live">{t('common:status.updating')}</BrandTag> : null}
    </ToolbarRow>
  );
}
