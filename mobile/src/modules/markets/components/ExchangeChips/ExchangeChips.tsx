import { Pressable, ScrollView, View } from 'react-native';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import type { ExchangeChipsProps } from './ExchangeChips.types';
import { styles } from './ExchangeChips.styles';

export function ExchangeChips({
  exchanges,
  selected,
  onSelect,
  isLoading,
}: ExchangeChipsProps) {
  if (isLoading && exchanges.length === 0) {
    return (
      <View style={styles.row}>
        <Skeleton width={80} height={32} borderRadius={999} />
        <Skeleton width={90} height={32} borderRadius={999} />
        <Skeleton width={70} height={32} borderRadius={999} />
      </View>
    );
  }

  return (
    <ScrollView horizontal showsHorizontalScrollIndicator={false}>
      <View style={styles.row}>
        {exchanges.map((exchange) => {
          const active = exchange === selected;
          return (
            <Pressable
              key={exchange}
              accessibilityRole="button"
              accessibilityState={{ selected: active }}
              onPress={() => onSelect(exchange)}
              style={[styles.chip, active && styles.chipActive]}
            >
              <Text
                variant="label"
                color={active ? 'cream' : 'secondary'}
                style={styles.chipLabel}
              >
                {exchange}
              </Text>
            </Pressable>
          );
        })}
      </View>
    </ScrollView>
  );
}
