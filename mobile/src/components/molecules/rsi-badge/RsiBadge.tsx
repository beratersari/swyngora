import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import type { RsiBadgeProps } from './RsiBadge.types';
import { styles } from './RsiBadge.styles';

export function RsiBadge({
  label,
  tone = 'secondary',
  loading = false,
  size = 'sm',
}: RsiBadgeProps) {
  return (
    <View style={[styles.wrap, loading ? styles.loading : null]} accessibilityLabel={label}>
      <Text variant={size === 'sm' ? 'caption' : 'body'} color={tone}>
        {label}
      </Text>
    </View>
  );
}
