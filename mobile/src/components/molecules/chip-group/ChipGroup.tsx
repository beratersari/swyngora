import { ScrollView, View } from 'react-native';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { Chip } from '@/components/molecules/chip';
import type { ChipGroupProps } from './ChipGroup.types';
import { styles } from './ChipGroup.styles';

function isSelected(selected: string | string[], value: string): boolean {
  return Array.isArray(selected) ? selected.includes(value) : selected === value;
}

export function ChipGroup({
  options,
  selected,
  onSelect,
  mode = 'single',
  shape = 'box',
  capitalizeLabels = false,
  horizontalScroll = false,
  isLoading = false,
  emptyLabel,
}: ChipGroupProps) {
  if (isLoading && options.length === 0) {
    return (
      <View style={styles.skeletonRow}>
        <Skeleton width={80} height={32} borderRadius={999} />
        <Skeleton width={90} height={32} borderRadius={999} />
        <Skeleton width={70} height={32} borderRadius={999} />
      </View>
    );
  }

  if (options.length === 0 && emptyLabel) {
    return (
      <Text variant="body" color="secondary">
        {emptyLabel}
      </Text>
    );
  }

  const chips = (
    <View style={styles.row}>
      {options.map((opt) => {
        const active = isSelected(selected, opt.value);
        return (
          <Chip
            key={opt.value}
            label={opt.label}
            active={active}
            shape={shape}
            onPress={() => onSelect(opt.value)}
            accessibilityRole={mode === 'multi' ? 'checkbox' : 'button'}
            accessibilityState={
              mode === 'multi' ? { checked: active } : { selected: active }
            }
          />
        );
      })}
    </View>
  );

  // capitalize via Text inside Chip — for exchange names we pass labels as-is
  void capitalizeLabels;

  if (horizontalScroll) {
    return (
      <ScrollView horizontal showsHorizontalScrollIndicator={false}>
        {chips}
      </ScrollView>
    );
  }

  return chips;
}
