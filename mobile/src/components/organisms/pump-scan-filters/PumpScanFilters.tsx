import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/text';
import { ChipGroup } from '@/components/molecules/chip-group';
import type { PumpScanFiltersProps } from './PumpScanFilters.types';
import { styles } from './PumpScanFilters.styles';

export function PumpScanFilters({
  exchanges,
  selectedExchange,
  onSelectExchange,
  exchangesLoading,
  lookbackHours,
  lookbackOptions,
  onSelectLookback,
  minReturnPct,
  thresholdOptions,
  onSelectThreshold,
  direction,
  directionOptions,
  onSelectDirection,
  summaryLabel,
}: PumpScanFiltersProps) {
  const { t } = useTranslation('pumps');
  return (
    <View style={styles.root}>
      <View style={styles.block}>
        <ChipGroup
          options={exchanges.map((e) => ({ value: e, label: e }))}
          selected={selectedExchange}
          onSelect={onSelectExchange}
          mode="single"
          shape="pill"
          horizontalScroll
          isLoading={exchangesLoading}
        />
      </View>

      <View style={styles.block}>
        <Text variant="caption" color="secondary">
          {t('lookback')}
        </Text>
        <ChipGroup
          options={lookbackOptions.map((h) => ({
            value: String(h),
            label: t('hours', { hours: h }),
          }))}
          selected={String(lookbackHours)}
          onSelect={(v) => onSelectLookback(Number(v))}
          mode="single"
          shape="box"
          horizontalScroll
        />
      </View>

      <View style={styles.block}>
        <Text variant="caption" color="secondary">
          {t('minReturn')}
        </Text>
        <ChipGroup
          options={thresholdOptions.map((p) => ({
            value: String(p),
            label: t('threshold', { pct: p }),
          }))}
          selected={String(minReturnPct)}
          onSelect={(v) => onSelectThreshold(Number(v))}
          mode="single"
          shape="box"
          horizontalScroll
        />
      </View>

      <View style={styles.block}>
        <Text variant="caption" color="secondary">
          {t('direction')}
        </Text>
        <ChipGroup
          options={directionOptions.map((d) => ({ value: d.value, label: d.label }))}
          selected={direction}
          onSelect={onSelectDirection}
          mode="single"
          shape="box"
        />
      </View>

      {summaryLabel ? (
        <Text variant="caption" color="steel">
          {summaryLabel}
        </Text>
      ) : null}
    </View>
  );
}
