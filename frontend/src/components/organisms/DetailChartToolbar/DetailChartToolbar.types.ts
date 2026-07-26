export type DetailChartToolbarProps = {
  intervals: string[];
  interval: string;
  limit: number;
  intervalsLoading?: boolean;
  onIntervalChange: (interval: string) => void;
  onLimitChange: (limit: number) => void;
  onRefresh?: () => void;
  isFetching?: boolean;
};
