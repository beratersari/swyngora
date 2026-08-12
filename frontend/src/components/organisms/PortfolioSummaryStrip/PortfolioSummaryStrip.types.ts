import type { PortfolioView } from '@/libs/api';

export type PortfolioSummaryStripProps = {
  view?: PortfolioView | null;
  isLoading?: boolean;
  currency?: string;
};
