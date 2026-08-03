import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { Chip } from '@/components/molecules/chip';
import type { ChatToolsChipsProps } from './ChatToolsChips.types';
import { styles } from './ChatToolsChips.styles';

export function ChatToolsChips({ tools, label }: ChatToolsChipsProps) {
  if (!tools.length) return null;
  return (
    <View style={styles.root} accessibilityRole="summary">
      {label ? (
        <Text variant="caption" color="secondary" style={styles.label}>
          {label}
        </Text>
      ) : null}
      {tools.map((tool) => (
        <Chip key={tool} label={tool} shape="box" active={false} />
      ))}
    </View>
  );
}
