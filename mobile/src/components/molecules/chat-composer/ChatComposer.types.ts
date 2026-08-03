export type ChatComposerProps = {
  value: string;
  onChangeText: (text: string) => void;
  onSend: () => void;
  placeholder?: string;
  sendLabel?: string;
  disabled?: boolean;
  /** Disable send only (input still editable). */
  sendDisabled?: boolean;
  accessibilityLabel?: string;
};
