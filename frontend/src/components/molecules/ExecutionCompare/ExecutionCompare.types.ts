import type { ExecutionCompare, ExecutionQuote } from '@/libs/api';
import type { ExecutionStyleId } from './constants';

export type ExecutionCompareProps = {
  /** Three prices for the current size. The page owns the request. */
  compare?: ExecutionCompare | null;
  /** True when the ticket already has a size. The empty line is for a blank quantity. */
  hasSize?: boolean;
  loading?: boolean;
  /** True while a newer read is in flight and the rows on screen are the previous success. */
  refreshing?: boolean;
  /** Client time of the last successful comparison, in milliseconds. */
  updatedAt?: number | null;
  failed?: boolean;
  /** The path the place button will send. */
  selected: ExecutionStyleId;
  minutes: number | null;
  slices: number | null;
  onSelect: (style: ExecutionStyleId) => void;
  onMinutes: (value: number | null) => void;
  onSlices: (value: number | null) => void;
};

export type { ExecutionQuote };
