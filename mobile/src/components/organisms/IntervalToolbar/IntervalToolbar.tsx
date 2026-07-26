import { View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { ChipGroup } from '@/components/molecules/ChipGroup';
import type { IntervalToolbarProps } from './IntervalToolbar.types';
import { styles } from './IntervalToolbar.styles';

export function IntervalToolbar({
  intervals,
  selected,
  onSelect,
  isLoading,
  showEma,
  onToggleEma,
}: IntervalToolbarProps) {
  return (
    <View style={styles.root}>
      <Text variant="label" color="secondary">
        Interval
      </Text>
      <View style={styles.row}>
        <ChipGroup
          options={intervals.map((i) => ({ value: i, label: i }))}
          selected={selected}
          onSelect={onSelect}
          mode="single"
          shape="box"
          horizontalScroll
          isLoading={isLoading}
        />
        <ChipGroup
          options={[{ value: 'ema', label: showEma ? 'EMA on' : 'EMA off' }]}
          selected={showEma ? 'ema' : ''}
          onSelect={onToggleEma}
          mode="single"
          shape="box"
        />
      </View>
    </View>
  );
}
