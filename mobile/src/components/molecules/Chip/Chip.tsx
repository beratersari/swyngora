import { Pressable } from 'react-native';
import { Text } from '@/components/atoms/Text';
import type { ChipProps } from './Chip.types';
import { styles } from './Chip.styles';

export function Chip({
  label,
  active = false,
  onPress,
  shape = 'pill',
  accessibilityRole = 'button',
  accessibilityState,
}: ChipProps) {
  return (
    <Pressable
      accessibilityRole={accessibilityRole}
      accessibilityState={accessibilityState ?? { selected: active }}
      onPress={onPress}
      style={[
        styles.base,
        shape === 'pill' ? styles.pill : styles.box,
        active && styles.active,
      ]}
    >
      <Text variant="caption" color={active ? 'cream' : 'secondary'}>
        {label}
      </Text>
    </Pressable>
  );
}
