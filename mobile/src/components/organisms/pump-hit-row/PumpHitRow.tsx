import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/text';
import type { PumpHitRowProps } from './PumpHitRow.types';
import { styles } from './PumpHitRow.styles';

export function PumpHitRow({ row, onPress }: PumpHitRowProps) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={() => onPress?.(row.exchange, row.symbol)}
      style={styles.row}
    >
      <View style={styles.top}>
        <Text variant="h4">{row.symbol}</Text>
        <Text variant="h4" color={row.bestReturnTone}>
          {row.bestReturnLabel}
        </Text>
      </View>
      <View style={styles.bottom}>
        <Text variant="caption" color="secondary" style={{ textTransform: 'capitalize' }}>
          {row.exchange}
        </Text>
        <Text variant="caption" color="steel">
          {row.eventsLabel}
        </Text>
      </View>
      {row.metaLabel ? (
        <Text variant="caption" color="steel" numberOfLines={1}>
          {row.metaLabel}
        </Text>
      ) : null}
    </Pressable>
  );
}
