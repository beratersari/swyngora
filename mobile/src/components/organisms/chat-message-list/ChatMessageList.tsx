import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { ChatBubble } from '@/components/molecules/chat-bubble';
import { ChatToolsChips } from '@/components/molecules/chat-tools-chips';
import type { ChatMessageListProps } from './ChatMessageList.types';
import { styles } from './ChatMessageList.styles';

export function ChatMessageList({
  messages,
  emptyTitle,
  emptyMessage,
  bannerError,
  toolsLabel,
}: ChatMessageListProps) {
  return (
    <View style={styles.list}>
      {bannerError ? (
        <Text variant="caption" color="error" style={styles.banner}>
          {bannerError}
        </Text>
      ) : null}

      {messages.length === 0 ? (
        <View style={styles.empty}>
          {emptyTitle ? (
            <Text variant="h3" color="primary">
              {emptyTitle}
            </Text>
          ) : null}
          {emptyMessage ? (
            <Text variant="body" color="secondary">
              {emptyMessage}
            </Text>
          ) : null}
        </View>
      ) : (
        messages.map((m) => (
          <View key={m.id}>
            <ChatBubble
              role={m.role}
              text={m.pending ? '' : m.text}
              pending={m.pending}
              error={m.error}
            />
            {m.tools && m.tools.length > 0 && !m.pending ? (
              <View style={styles.tools}>
                <ChatToolsChips tools={m.tools} label={toolsLabel} />
              </View>
            ) : null}
          </View>
        ))
      )}
    </View>
  );
}
