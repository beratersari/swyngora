import type { SpotMetricDef, SpotMetricId } from '@/libs/utils/spotMetrics';

export type MetricColumnPickerProps = {
  /** All metrics the user may toggle for this surface. */
  available: SpotMetricDef[];
  /** Currently selected metric ids (order = column order). */
  value: SpotMetricId[];
  onChange: (ids: SpotMetricId[]) => void;
  onReset?: () => void;
  /** i18n label resolver: metric labelKey → display title */
  getLabel: (labelKey: string) => string;
  ariaLabel?: string;
  resetLabel?: string;
  buttonLabel?: string;
  moveUpLabel?: string;
  moveDownLabel?: string;
  dragHintLabel?: string;
};
