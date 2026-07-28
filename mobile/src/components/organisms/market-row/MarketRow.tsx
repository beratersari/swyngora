import { Pressable, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/text';
import { RsiBadge } from '@/components/molecules/rsi-badge';
import { StarButton } from '@/components/molecules/star-button';
import type { MarketRowProps } from './MarketRow.types';
import { styles } from './MarketRow.styles';

export function MarketRow({ row, onPress, watched, onStarPress }: MarketRowProps) {
  const { t } = useTranslation(['common', 'markets']);
  return (
    <View style={styles.row}>
      {onStarPress != null ? (
        <StarButton
          watched={Boolean(watched)}
          size="sm"
          onPress={() => onStarPress(row.symbol)}
          accessibilityLabel={
            watched
              ? t('common:a11y.removeFavorite', { symbol: row.symbol })
              : t('common:a11y.addFavorite', { symbol: row.symbol })
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
          <Text variant="caption" color={row.changeTone} style={styles.meta}>
            {row.changePercentLabel}
          </Text>
          <Text variant="caption" color="secondary">
            {t('markets:vol', { value: row.quoteVolumeLabel })}
          </Text>
          <Text variant="caption" color="secondary">
            {t('markets:mcap', { value: row.marketCapLabel })}
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
