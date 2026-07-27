import { Pressable, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/text';
import { RsiBadge } from '@/components/molecules/rsi-badge';
import { StarButton } from '@/components/molecules/star-button';
import type { WatchlistRowProps } from './WatchlistRow.types';
import { styles } from './WatchlistRow.styles';

export function WatchlistRow({ row, onPress, onUnstar }: WatchlistRowProps) {
  const { t } = useTranslation('common');
  return (
    <View style={styles.row}>
      <Pressable
        accessibilityRole="button"
        onPress={() => onPress?.(row.exchange, row.symbol)}
        style={styles.main}
      >
        <View style={styles.top}>
          <Text variant="h4">{row.symbol}</Text>
          <View style={styles.topRight}>
            {row.rsiLabel != null ? (
              <RsiBadge
                label={row.rsiLabel}
                tone={row.rsiTone ?? 'secondary'}
                loading={row.rsiLoading}
                size="sm"
              />
            ) : null}
            <Text variant="numeric">{row.lastPriceLabel}</Text>
          </View>
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
        accessibilityLabel={t('a11y.removeFavorite', { symbol: row.symbol })}
      />
    </View>
  );
}
