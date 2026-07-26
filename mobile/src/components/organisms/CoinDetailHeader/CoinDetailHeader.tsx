import { Pressable, View } from 'react-native';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
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
}: CoinDetailHeaderProps) {
  return (
    <View style={styles.card}>
      <Pressable onPress={onBack} accessibilityRole="button">
        <Text variant="caption" color="steel">
          ← Markets
        </Text>
      </Pressable>
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
