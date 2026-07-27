import { Button, Select, Switch, Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
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
}: DetailChartToolbarProps) {
  const { t } = useTranslation(['detail', 'common']);

  const options =
    intervals.length > 0
      ? intervals.map((iv) => ({ value: iv, label: iv }))
      : [{ value: interval, label: interval }];

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
        <Select
          size="small"
          value={interval}
          options={options}
          loading={intervalsLoading}
          onChange={onIntervalChange}
          style={{ minWidth: 100 }}
          showSearch
          optionFilterProp="label"
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

      {onRefresh ? (
        <Button size="small" onClick={onRefresh} loading={isFetching}>
          {t('common:actions.refresh')}
        </Button>
      ) : null}
      {isFetching ? <Tag color="processing">{t('common:status.updating')}</Tag> : null}
    </ToolbarRow>
  );
}
