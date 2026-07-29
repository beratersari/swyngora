import type { CrossExchangeRowModel } from '@/libs/utils';

export type CrossExchangeCompareProps = {
  title: string;
  rows: CrossExchangeRowModel[];
  disclaimer?: string | null;
  emptyMessage?: string | null;
  cheapestId?: string | null;
  unavailableLabel?: string;
  sourceLabel?: string;
  cheapestLabel?: string;
  onPressRow?: (exchange: string, symbol: string) => void;
};
