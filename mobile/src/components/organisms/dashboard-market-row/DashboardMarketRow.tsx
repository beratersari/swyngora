import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/text';
import type { DashboardMarketRowProps } from './DashboardMarketRow.types';
import { styles } from './DashboardMarketRow.styles';

export function DashboardMarketRow({ row, onPress }: DashboardMarketRowProps) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={() => onPress?.(row.exchange, row.symbol)}
      style={styles.row}
    >
      <View style={styles.left}>
        <Text variant="h4">{row.symbol}</Text>
        <Text variant="caption" color="steel" style={styles.exchange}>
          {row.exchange}
          {row.metaLabel ? ` · ${row.metaLabel}` : ''}
        </Text>
      </View>
      <View style={styles.right}>
        <Text variant="numeric">{row.lastPriceLabel}</Text>
        <Text variant="caption" color={row.changeTone}>
          {row.changePercentLabel}
        </Text>
      </View>
    </Pressable>
  );
}
