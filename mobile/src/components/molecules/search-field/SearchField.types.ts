export type SearchFieldProps = {
  value: string;
  onChangeText: (value: string) => void;
  placeholder?: string;
  accessibilityLabel?: string;
  autoCapitalize?: 'none' | 'sentences' | 'words' | 'characters';
  autoCorrect?: boolean;
  style?: object;
};
