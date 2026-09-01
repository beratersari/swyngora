import type { TickerTapeItem } from '@/libs/utils';

export type { TickerTapeItem };

export type TickerTapeProps = {
  items: TickerTapeItem[];
  ariaLabel: string;
  /** Pause the marquee (hidden tab) to cut compositor work. */
  paused?: boolean;
};
