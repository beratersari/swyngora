import type { DashboardMarketRowData } from '@/components/organisms/dashboard-market-row';

export type DashboardSectionListProps = {
  title: string;
  actionLabel?: string;
  onAction?: () => void;
  rows: DashboardMarketRowData[];
  isLoading?: boolean;
  errorMessage?: string | null;
  emptyMessage?: string | null;
  onPressRow?: (exchange: string, symbol: string) => void;
  onRetry?: () => void;
  retryLabel?: string;
};
