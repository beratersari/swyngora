export type SplitBarPart = {
  key: string;
  /** Share of the bar. Parts are scaled to fill the track. */
  pct: number;
  color: string;
};

export type SplitBarProps = {
  parts: SplitBarPart[];
  ariaLabel: string;
};
