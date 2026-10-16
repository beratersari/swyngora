import type { ReasonHistoryRow } from './helpers';

export type OrderReasonHistoryProps = {
  rows?: readonly ReasonHistoryRow[] | null;
  selected: readonly string[];
};
