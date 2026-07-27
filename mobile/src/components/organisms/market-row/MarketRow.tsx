import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { StarButton } from '@/components/molecules/star-button';
import type { MarketRowProps } from './MarketRow.types';
import { styles } from './MarketRow.styles';

export function MarketRow({ row, onPress, watched, onStarPress }: MarketRowProps) {
  return (
    <View style={styles.row}>
      {onStarPress != null ? (
        <StarButton
          watched={Boolean(watched)}
          size="sm"
          onPress={() => onStarPress(row.symbol)}
          accessibilityLabel={
            watched ? `Remove ${row.symbol} from favorites` : `Add ${row.symbol} to favorites`
          }
        />
      ) : null}
      <Pressable
        accessibilityRole="button"
        onPress={() => onPress?.(row.symbol)}
        style={styles.main}
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
    </View>
  );
}
