import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { StatTile } from '@/components/molecules/stat-tile';
import type { CoinDetailStatsProps } from './CoinDetailStats.types';
import { styles } from './CoinDetailStats.styles';

export function CoinDetailStats({
  items,
  isLoading,
  tickerError,
  supplyError,
}: CoinDetailStatsProps) {
  return (
    <View style={styles.section}>
      {tickerError ? (
        <Text variant="caption" color="error">
          Ticker: {tickerError}
        </Text>
      ) : null}
      {supplyError ? (
        <Text variant="caption" color="steel">
          Supply: {supplyError}
        </Text>
      ) : null}
      <View style={styles.grid}>
        {items.map((item) => (
          <StatTile
            key={item.label}
            label={item.label}
            value={item.value}
            isLoading={isLoading}
          />
        ))}
      </View>
    </View>
  );
}
