import { Pressable, View } from 'react-native';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { StarButton } from '@/components/molecules/star-button';
import type { CoinDetailHeaderProps } from './CoinDetailHeader.types';
import { styles } from './CoinDetailHeader.styles';

export function CoinDetailHeader({
  symbol,
  exchange,
  lastPriceLabel,
  changePercentLabel,
  changeTone,
  isLoading,
  onBack,
  watched,
  onStarPress,
}: CoinDetailHeaderProps) {
  return (
    <View style={styles.card}>
      <View style={styles.topBar}>
        <Pressable onPress={onBack} accessibilityRole="button">
          <Text variant="caption" color="steel">
            ← Back
          </Text>
        </Pressable>
        {onStarPress != null ? (
          <StarButton
            watched={Boolean(watched)}
            onPress={onStarPress}
            accessibilityLabel={
              watched ? `Remove ${symbol} from favorites` : `Add ${symbol} to favorites`
            }
          />
        ) : null}
      </View>
      <View style={styles.top}>
        <View style={styles.left}>
          {isLoading ? (
            <Skeleton height={28} width={140} />
          ) : (
            <Text variant="h2">{symbol}</Text>
          )}
          <Text variant="caption" color="secondary" style={styles.exchange}>
            {exchange}
          </Text>
        </View>
        <View style={styles.right}>
          {isLoading ? (
            <Skeleton height={24} width={100} />
          ) : (
            <Text variant="h3">{lastPriceLabel}</Text>
          )}
          {isLoading ? (
            <Skeleton height={16} width={72} />
          ) : (
            <Text variant="label" color={changeTone}>
              {changePercentLabel}
            </Text>
          )}
        </View>
      </View>
    </View>
  );
}
