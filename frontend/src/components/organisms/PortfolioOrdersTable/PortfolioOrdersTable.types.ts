import type { PendingOrder } from '@/libs/api';

export type PortfolioOrdersTableProps = {
  items: PendingOrder[];
  loading?: boolean;
  cancelLoading?: boolean;
  onCancel?: (id: string) => void;
  onOpen?: (exchange: string, symbol: string) => void;
};
