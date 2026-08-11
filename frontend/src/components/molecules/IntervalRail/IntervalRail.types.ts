export type IntervalRailProps = {
  intervals: string[];
  value: string;
  onChange: (interval: string) => void;
  loading?: boolean;
  'aria-label'?: string;
};
