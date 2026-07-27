export type ChipProps = {
  label: string;
  active?: boolean;
  onPress?: () => void;
  /** pill (default) or box */
  shape?: 'pill' | 'box';
  accessibilityRole?: 'button' | 'checkbox';
  accessibilityState?: { selected?: boolean; checked?: boolean };
};
