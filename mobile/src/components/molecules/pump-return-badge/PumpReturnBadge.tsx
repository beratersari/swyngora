import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { formatPumpReturnPct, pumpReturnTone } from '@/libs/utils';
import type { PumpReturnBadgeProps } from './PumpReturnBadge.types';
import { styles } from './PumpReturnBadge.styles';

export function PumpReturnBadge({ returnPct, size = 'md' }: PumpReturnBadgeProps) {
  const tone = pumpReturnTone(returnPct);
  return (
    <View style={styles.wrap}>
      <Text variant={size === 'sm' ? 'caption' : 'h4'} color={tone}>
        {formatPumpReturnPct(returnPct)}
      </Text>
    </View>
  );
}
