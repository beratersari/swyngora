import type { MarketFundingArbScan } from '@/libs/api';

export type FundingArbScanTableProps = {
  scan?: MarketFundingArbScan | null;
  isLoading?: boolean;
};
