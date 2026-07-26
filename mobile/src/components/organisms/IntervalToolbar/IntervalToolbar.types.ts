export type IntervalToolbarProps = {
  intervals: string[];
  selected: string;
  onSelect: (interval: string) => void;
  isLoading?: boolean;
  showEma: boolean;
  onToggleEma: () => void;
};
