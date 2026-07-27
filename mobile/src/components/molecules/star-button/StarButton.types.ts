export type StarButtonProps = {
  watched: boolean;
  onPress: () => void;
  disabled?: boolean;
  /** Accessible name override */
  accessibilityLabel?: string;
  size?: 'sm' | 'md';
};
