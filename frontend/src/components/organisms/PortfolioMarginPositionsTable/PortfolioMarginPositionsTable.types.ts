import type { MarginPosition } from '@/libs/api';

export type PortfolioMarginPositionsTableProps = {
  items: MarginPosition[];
  loading?: boolean;
  closeLoading?: boolean;
  onClose?: (id: string, quantity?: number) => Promise<void> | void;
  onOpen?: (exchange: string, symbol: string) => void;
};
