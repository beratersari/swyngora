import { Pressable, View } from 'react-native';
import { Text } from '@/components/atoms/text';
import type { SectionHeaderProps } from './SectionHeader.types';
import { styles } from './SectionHeader.styles';

export function SectionHeader({ title, actionLabel, onAction }: SectionHeaderProps) {
  return (
    <View style={styles.row}>
      <Text variant="label" color="secondary">
        {title}
      </Text>
      {actionLabel && onAction ? (
        <Pressable onPress={onAction} accessibilityRole="button">
          <Text variant="caption" color="cream">
            {actionLabel}
          </Text>
        </Pressable>
      ) : null}
    </View>
  );
}
