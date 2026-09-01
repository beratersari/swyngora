import type { ReactNode } from 'react';
import { DESK_TAPE_SOURCES, type DeskTapeSource, type TickerTapeItem } from '@/libs/utils';

export { DESK_TAPE_SOURCES };
export type { DeskTapeSource };

export type DeskPriceTapeProps = {
  source: DeskTapeSource;
  onSourceChange: (next: DeskTapeSource) => void;
  items: TickerTapeItem[];
  isLoading?: boolean;
  emptyLabel?: string;
  sourceAriaLabel: string;
  tapeAriaLabel: string;
  paused?: boolean;
  /** Custom tape body (e.g. per-symbol watchlist quotes). */
  children?: ReactNode;
};
