export type TickerTapeItem = {
  exchange: string;
  symbol: string;
  lastPrice: string;
  changePercent: string;
  changeValue: number | null;
  href: string;
};

export type TickerTapeProps = {
  items: TickerTapeItem[];
  ariaLabel: string;
  /** Pause the marquee (hidden tab) to cut compositor work. */
  paused?: boolean;
};
