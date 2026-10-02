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
  showDelta = true,
  onShowDeltaChange,
  showCvd = true,
  onShowCvdChange,
  showLiqLevels = true,
  onShowLiqLevelsChange,
  showBook = true,
  onShowBookChange,
  showFootprint = true,
  onShowFootprintChange,
  showCrowd = true,
  onShowCrowdChange,
  showOi = true,
  onShowOiChange,
  showFunding = true,
  onShowFundingChange,
  showLiqNotional = true,
  onShowLiqNotionalChange,
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

      {onShowDeltaChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.delta')}
          </Text>
          <Switch
            size="small"
            checked={showDelta}
            onChange={onShowDeltaChange}
            aria-label={t('detail:chart.delta')}
          />
        </InlineField>
      ) : null}

      {onShowCvdChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.cvd')}
          </Text>
          <Switch
            size="small"
            checked={showCvd}
            onChange={onShowCvdChange}
            aria-label={t('detail:chart.cvd')}
          />
        </InlineField>
      ) : null}

      {onShowLiqLevelsChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.liqLevels')}
          </Text>
          <Switch
            size="small"
            checked={showLiqLevels}
            onChange={onShowLiqLevelsChange}
            aria-label={t('detail:chart.liqLevels')}
          />
        </InlineField>
      ) : null}

      {onShowBookChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.book')}
          </Text>
          <Switch
            size="small"
            checked={showBook}
            onChange={onShowBookChange}
            aria-label={t('detail:chart.book')}
          />
        </InlineField>
      ) : null}

      {onShowFootprintChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.footprint')}
          </Text>
          <Switch
            size="small"
            checked={showFootprint}
            onChange={onShowFootprintChange}
            aria-label={t('detail:chart.footprint')}
          />
        </InlineField>
      ) : null}

      {onShowCrowdChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.crowd')}
          </Text>
          <Switch
            size="small"
            checked={showCrowd}
            onChange={onShowCrowdChange}
            aria-label={t('detail:chart.crowd')}
          />
        </InlineField>
      ) : null}

      {onShowOiChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.oi')}
          </Text>
          <Switch
            size="small"
            checked={showOi}
            onChange={onShowOiChange}
            aria-label={t('detail:chart.oi')}
          />
        </InlineField>
      ) : null}

      {onShowFundingChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.funding')}
          </Text>
          <Switch
            size="small"
            checked={showFunding}
            onChange={onShowFundingChange}
            aria-label={t('detail:chart.funding')}
          />
        </InlineField>
      ) : null}

      {onShowLiqNotionalChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.liqNotional')}
          </Text>
          <Switch
            size="small"
            checked={showLiqNotional}
            onChange={onShowLiqNotionalChange}
            aria-label={t('detail:chart.liqNotional')}
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
