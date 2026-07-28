import { View, TextInput } from 'react-native';
import { Button } from '@/components/atoms/button';
import { semanticColors } from '@/styles/tokens';
import type { ChatComposerProps } from './ChatComposer.types';
import { styles } from './ChatComposer.styles';

export function ChatComposer({
  value,
  onChangeText,
  onSend,
  placeholder = 'Ask about markets…',
  sendLabel = 'Send',
  disabled = false,
  sendDisabled = false,
  accessibilityLabel = 'Chat message',
}: ChatComposerProps) {
  const canSend = !disabled && !sendDisabled && value.trim().length > 0;

  return (
    <View style={styles.row}>
      <TextInput
        accessibilityLabel={accessibilityLabel}
        placeholder={placeholder}
        placeholderTextColor={semanticColors.text.disabled}
        value={value}
        onChangeText={onChangeText}
        editable={!disabled}
        multiline
        style={styles.input}
        onSubmitEditing={() => {
          if (canSend) onSend();
        }}
      />
      <View style={styles.sendWrap}>
        <Button
          label={sendLabel}
          onPress={onSend}
          disabled={!canSend}
          variant="primary"
        />
      </View>
    </View>
  );
}
