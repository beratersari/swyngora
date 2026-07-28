import { View } from 'react-native';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import type { ChatBubbleProps } from './ChatBubble.types';
import { styles } from './ChatBubble.styles';

export function ChatBubble({
  role,
  text,
  pending = false,
  error,
  metaLabels,
}: ChatBubbleProps) {
  const isUser = role === 'user';
  const isSystem = role === 'system';

  return (
    <View
      style={[
        styles.row,
        isUser ? styles.rowUser : styles.rowAssistant,
      ]}
      accessibilityRole="text"
    >
      <View
        style={[
          styles.bubble,
          isUser
            ? styles.bubbleUser
            : isSystem
              ? styles.bubbleSystem
              : styles.bubbleAssistant,
        ]}
      >
        {pending ? (
          <Skeleton height={16} width={120} />
        ) : (
          <Text
            variant={isSystem ? 'caption' : 'body'}
            color={isSystem ? 'secondary' : 'primary'}
          >
            {text}
          </Text>
        )}
        {metaLabels && metaLabels.length > 0 ? (
          <Text variant="caption" color="steel" style={styles.meta}>
            {metaLabels.join(' · ')}
          </Text>
        ) : null}
        {error ? (
          <Text variant="caption" color="error" style={styles.error}>
            {error}
          </Text>
        ) : null}
      </View>
    </View>
  );
}
