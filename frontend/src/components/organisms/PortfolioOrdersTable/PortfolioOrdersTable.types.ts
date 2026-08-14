import type { PendingOrder } from '@/libs/api';

export type AmendOrderValues = {
  id: string;
  triggerPrice?: number;
  remainingQuantity?: number;
};

export type PortfolioOrdersTableProps = {
  items: PendingOrder[];
  loading?: boolean;
  cancelLoading?: boolean;
  amendLoading?: boolean;
  onCancel?: (id: string) => void;
  onAmend?: (values: AmendOrderValues) => Promise<void>;
  onOpen?: (exchange: string, symbol: string) => void;
};
