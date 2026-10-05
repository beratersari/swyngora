import { Button, Segmented, Select, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { IntervalRail } from '@/components/molecules/IntervalRail';
import { LIQ_HEATMAP_RANGES } from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.constants';
import { DETAIL_PUMP_THRESHOLD_OPTIONS } from '@/config/constants';
import { Field, InlineField, ToolbarRow } from './DetailChartToolbar.styles';
import type { DetailChartToolbarProps } from './DetailChartToolbar.types';

const CVD_RANGES = ['24h', '7d', '30d', '90d'] as const;
const LIQ_MAP_VENUES = ['combined', 'binance', 'bybit', 'okx'] as const;
const LIQ_MAP_SIDES = ['totals', 'longs', 'shorts'] as const;
const LIQ_MAP_SOURCES = ['prints', 'model'] as const;

export function DetailChartToolbar({
  chartPreset = null,
  presetOptions,
  onChartPresetChange,
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
  cvdRange = '24h',
  onCvdRangeChange,
  showLiqLevels = true,
  onShowLiqLevelsChange,
  showBook = true,
  onShowBookChange,
  showFootprint = true,
  onShowFootprintChange,
  showProfile = true,
  onShowProfileChange,
  showSweeps = true,
  onShowSweepsChange,
  showBots = true,
  onShowBotsChange,
  showCrowd = true,
  onShowCrowdChange,
  showOi = true,
  onShowOiChange,
  showFunding = true,
  onShowFundingChange,
  showLiqNotional = true,
  onShowLiqNotionalChange,
  showLiqMap = true,
  onShowLiqMapChange,
  showOpenLevels = true,
  onShowOpenLevelsChange,
  liqMapRange = '24h',
  onLiqMapRangeChange,
  liqMapVenue = 'combined',
  onLiqMapVenueChange,
  liqMapSide = 'totals',
  onLiqMapSideChange,
  liqMapSource = 'prints',
  onLiqMapSourceChange,
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
      {onChartPresetChange && presetOptions && presetOptions.length > 0 ? (
        // A label wrapper would name and activate only the first desk.
        <Field as="div">
          <Text variant="caption" color="secondary">
            {t('detail:chart.preset')}
          </Text>
          <Segmented
            size="small"
            aria-label={t('detail:chart.preset')}
            value={chartPreset ?? ''}
            options={presetOptions}
            onChange={(value) => onChartPresetChange(String(value))}
          />
        </Field>
      ) : null}
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

      {onCvdRangeChange ? (
        <Field $compact>
          <Text variant="caption" color="secondary">
            {t('detail:chart.cvdRange')}
          </Text>
          <Select
            size="small"
            value={cvdRange}
            options={CVD_RANGES.map((range) => ({
              value: range,
              label: t(`detail:chart.cvdRanges.${range}`),
            }))}
            onChange={onCvdRangeChange}
            style={{ minWidth: 88 }}
            aria-label={t('detail:chart.cvdRange')}
          />
        </Field>
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

      {onShowProfileChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.profile')}
          </Text>
          <Switch
            size="small"
            checked={showProfile}
            onChange={onShowProfileChange}
            aria-label={t('detail:chart.profile')}
          />
        </InlineField>
      ) : null}

      {onShowSweepsChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.sweeps')}
          </Text>
          <Switch
            size="small"
            checked={showSweeps}
            onChange={onShowSweepsChange}
            aria-label={t('detail:chart.sweeps')}
          />
        </InlineField>
      ) : null}

      {onShowBotsChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.bots')}
          </Text>
          <Switch
            size="small"
            checked={showBots}
            onChange={onShowBotsChange}
            aria-label={t('detail:chart.bots')}
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

      {onShowLiqMapChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.liqMap')}
          </Text>
          <Switch
            size="small"
            checked={showLiqMap}
            onChange={onShowLiqMapChange}
            aria-label={t('detail:chart.liqMap')}
          />
        </InlineField>
      ) : null}

      {onShowOpenLevelsChange ? (
        <InlineField>
          <Text variant="caption" color="secondary">
            {t('detail:chart.openLevels')}
          </Text>
          <Switch
            size="small"
            checked={showOpenLevels}
            onChange={onShowOpenLevelsChange}
            aria-label={t('detail:chart.openLevels')}
          />
        </InlineField>
      ) : null}

      {(showLiqMap || showOpenLevels) && onLiqMapRangeChange ? (
        <Field $compact>
          <Text variant="caption" color="secondary">
            {t('detail:liqHeatmap.range')}
          </Text>
          <Select
            size="small"
            value={liqMapRange}
            options={LIQ_HEATMAP_RANGES.map((range) => ({
              value: range,
              label: t(`detail:liqHeatmap.ranges.${range}`),
            }))}
            onChange={onLiqMapRangeChange}
            style={{ minWidth: 88 }}
            aria-label={t('detail:liqHeatmap.range')}
          />
        </Field>
      ) : null}

      {(showLiqMap || showOpenLevels) && onLiqMapVenueChange ? (
        <Field $compact>
          <Text variant="caption" color="secondary">
            {t('detail:liqHeatmap.venue')}
          </Text>
          <Select
            size="small"
            value={liqMapVenue}
            options={LIQ_MAP_VENUES.map((venue) => ({
              value: venue,
              label: t(`detail:liqHeatmap.venues.${venue}`),
            }))}
            onChange={onLiqMapVenueChange}
            style={{ minWidth: 110 }}
            aria-label={t('detail:liqHeatmap.venue')}
          />
        </Field>
      ) : null}

      {onShowLiqMapChange && showLiqMap && onLiqMapSideChange ? (
        <Field $compact>
          <Text variant="caption" color="secondary">
            {t('detail:liqHeatmap.side')}
          </Text>
          <Select
            size="small"
            value={liqMapSide}
            options={LIQ_MAP_SIDES.map((side) => ({
              value: side,
              label: t(`detail:liqHeatmap.sides.${side}`),
            }))}
            onChange={onLiqMapSideChange}
            style={{ minWidth: 96 }}
            aria-label={t('detail:liqHeatmap.side')}
          />
        </Field>
      ) : null}

      {onShowLiqMapChange && showLiqMap && onLiqMapSourceChange ? (
        <Field $compact>
          <Text variant="caption" color="secondary">
            {t('detail:liqHeatmap.source')}
          </Text>
          <Select
            size="small"
            value={liqMapSource}
            options={LIQ_MAP_SOURCES.map((id) => ({
              value: id,
              label: t(`detail:liqHeatmap.sources.${id}`),
            }))}
            onChange={onLiqMapSourceChange}
            style={{ minWidth: 96 }}
            aria-label={t('detail:liqHeatmap.sourceLabel')}
          />
        </Field>
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
