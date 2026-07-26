import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import type { MarketRowProps } from './MarketRow.types';
import { styles } from './MarketRow.styles';

export function MarketRow({ row, onPress }: MarketRowProps) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={() => onPress?.(row.symbol)}
      style={styles.row}
    >
      <View style={styles.top}>
        <Text variant="h4">{row.symbol}</Text>
        <Text variant="numeric">{row.lastPriceLabel}</Text>
      </View>
      <View style={styles.bottom}>
        <Text variant="caption" color={row.changeTone} style={styles.meta}>
          {row.changePercentLabel}
        </Text>
        <Text variant="caption" color="secondary">
          Vol {row.quoteVolumeLabel}
        </Text>
        <Text variant="caption" color="secondary">
          Mcap {row.marketCapLabel}
        </Text>
      </View>
      {row.tagsLabel ? (
        <Text variant="caption" color="steel" numberOfLines={1}>
          {row.tagsLabel}
        </Text>
      ) : null}
    </Pressable>
  );
}
