import { Pressable } from 'react-native';
import { Text } from '@/components/atoms/Text';
import type { ButtonProps } from './Button.types';
import { styles } from './Button.styles';

export function Button({
  label,
  onPress,
  disabled = false,
  variant = 'primary',
}: ButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      disabled={disabled}
      onPress={onPress}
      style={[
        styles.base,
        variant === 'primary' ? styles.primary : styles.secondary,
        disabled && styles.disabled,
      ]}
    >
      <Text
        style={[styles.label, variant === 'secondary' && styles.labelSecondary]}
        variant="label"
      >
        {label}
      </Text>
    </Pressable>
  );
}
