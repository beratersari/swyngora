import { TextInput } from 'react-native';
import { semanticColors } from '@/styles/tokens';
import type { SearchFieldProps } from './SearchField.types';
import { styles } from './SearchField.styles';

export function SearchField({
  value,
  onChangeText,
  placeholder = 'Search…',
  accessibilityLabel = 'Search',
  autoCapitalize = 'none',
  autoCorrect = false,
  style,
}: SearchFieldProps) {
  return (
    <TextInput
      accessibilityLabel={accessibilityLabel}
      placeholder={placeholder}
      placeholderTextColor={semanticColors.text.disabled}
      value={value}
      onChangeText={onChangeText}
      autoCapitalize={autoCapitalize}
      autoCorrect={autoCorrect}
      style={[styles.input, style]}
    />
  );
}
