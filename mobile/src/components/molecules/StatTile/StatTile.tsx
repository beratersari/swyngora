import { View } from 'react-native';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import type { StatTileProps } from './StatTile.types';
import { styles } from './StatTile.styles';

export function StatTile({ label, value, isLoading }: StatTileProps) {
  return (
    <View style={styles.tile}>
      <Text variant="caption" color="secondary">
        {label}
      </Text>
      {isLoading ? (
        <Skeleton height={18} width="70%" />
      ) : (
        <Text variant="numeric">{value}</Text>
      )}
    </View>
  );
}
