import type { ReactNode } from 'react';
import type { TickerTapeItem } from '@/components/molecules/TickerTape';

export const DESK_TAPE_SOURCES = ['binance', 'coinbase', 'bist', 'watchlist'] as const;

export type DeskTapeSource = (typeof DESK_TAPE_SOURCES)[number];

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
