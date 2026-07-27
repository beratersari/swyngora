import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { StarButton } from '@/components/molecules/StarButton';
import type { WatchlistRowProps } from './WatchlistRow.types';
import { styles } from './WatchlistRow.styles';

export function WatchlistRow({ row, onPress, onUnstar }: WatchlistRowProps) {
  return (
    <View style={styles.row}>
      <Pressable
        accessibilityRole="button"
        onPress={() => onPress?.(row.exchange, row.symbol)}
        style={styles.main}
      >
        <View style={styles.top}>
          <Text variant="h4">{row.symbol}</Text>
          <Text variant="numeric">{row.lastPriceLabel}</Text>
        </View>
        <View style={styles.bottom}>
          <Text variant="caption" color="secondary" style={{ textTransform: 'capitalize' }}>
            {row.exchange}
          </Text>
          <Text variant="caption" color={row.changeTone}>
            {row.changePercentLabel}
          </Text>
        </View>
      </Pressable>
      <StarButton
        watched
        size="sm"
        onPress={() => onUnstar?.(row.exchange, row.symbol)}
        accessibilityLabel={`Remove ${row.symbol} from favorites`}
      />
    </View>
  );
}
