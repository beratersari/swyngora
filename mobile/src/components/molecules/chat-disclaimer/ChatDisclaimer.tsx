import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import type { ChatDisclaimerProps } from './ChatDisclaimer.types';
import { styles } from './ChatDisclaimer.styles';

export function ChatDisclaimer({ text }: ChatDisclaimerProps) {
  return (
    <View style={styles.root} accessibilityRole="text">
      <Text variant="caption" color="steel">
        {text}
      </Text>
    </View>
  );
}
