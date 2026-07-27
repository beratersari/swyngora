export type ChipOption = {
  value: string;
  label: string;
};

export type ChipGroupProps = {
  options: ChipOption[];
  selected: string | string[];
  onSelect: (value: string) => void;
  /** single: radio-like; multi: checkbox-like */
  mode?: 'single' | 'multi';
  shape?: 'pill' | 'box';
  capitalizeLabels?: boolean;
  horizontalScroll?: boolean;
  isLoading?: boolean;
  emptyLabel?: string;
};
