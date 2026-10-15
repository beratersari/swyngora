import type { MarginPosition } from '@/libs/api';

/** Fields sent to PUT /margin/positions/{id}/brackets. Omitted fields stay as they are. */
export type MarginBracketUpdate = {
  id: string;
  stopLoss?: number;
  takeProfit?: number;
  clearStopLoss?: boolean;
  clearTakeProfit?: boolean;
};

export type PortfolioMarginPositionsTableProps = {
  items: MarginPosition[];
  loading?: boolean;
  closeLoading?: boolean;
  bracketsLoading?: boolean;
  onClose?: (id: string, quantity?: number) => Promise<void> | void;
  onBrackets?: (values: MarginBracketUpdate) => Promise<void>;
  onOpen?: (exchange: string, symbol: string) => void;
};
