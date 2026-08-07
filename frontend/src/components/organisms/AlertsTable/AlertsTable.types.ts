import type { PriceAlert } from '@/libs/api/endpoints/alertsApi';

export type AlertsTableProps = {
  items: PriceAlert[];
  loading?: boolean;
  deleteLoading?: boolean;
  onDelete: (id: string) => void;
  onOpen?: (exchange: string, symbol: string) => void;
};
