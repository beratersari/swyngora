import { Button, Select, Tag } from 'antd';
import { Text } from '@/components/atoms/Text';
import { CANDLE_LIMIT_OPTIONS } from './DetailChartToolbar.constants';
import { Field, ToolbarRow } from './DetailChartToolbar.styles';
import type { DetailChartToolbarProps } from './DetailChartToolbar.types';

export function DetailChartToolbar({
  intervals,
  interval,
  limit,
  intervalsLoading,
  onIntervalChange,
  onLimitChange,
  onRefresh,
  isFetching,
}: DetailChartToolbarProps) {
  const options =
    intervals.length > 0
      ? intervals.map((iv) => ({ value: iv, label: iv }))
      : [{ value: interval, label: interval }];

  return (
    <ToolbarRow>
      <Field>
        <Text variant="caption" color="steel">
          Interval
        </Text>
        <Select
          value={interval}
          options={options}
          loading={intervalsLoading}
          onChange={onIntervalChange}
          style={{ minWidth: 120 }}
          showSearch
          optionFilterProp="label"
        />
      </Field>
      <Field>
        <Text variant="caption" color="steel">
          Bars
        </Text>
        <Select
          value={limit}
          options={CANDLE_LIMIT_OPTIONS.map((n) => ({ value: n, label: String(n) }))}
          onChange={onLimitChange}
          style={{ minWidth: 100 }}
        />
      </Field>
      {onRefresh ? (
        <Button onClick={onRefresh} loading={isFetching}>
          Refresh
        </Button>
      ) : null}
      {isFetching ? <Tag color="processing">updating…</Tag> : null}
    </ToolbarRow>
  );
}
