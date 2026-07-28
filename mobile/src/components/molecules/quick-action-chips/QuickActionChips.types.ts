export type QuickActionItem = {
  id: string;
  label: string;
  onPress: () => void;
};

export type QuickActionChipsProps = {
  actions: QuickActionItem[];
};
