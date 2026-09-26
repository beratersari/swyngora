import type { MarketFundingArbHistory } from '@/libs/api';

export type FundingArbHistoryTableProps = {
  history?: MarketFundingArbHistory | null;
  isLoading?: boolean;
};
