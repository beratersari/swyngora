import type { MarketOptionsExpiry } from '@/libs/api';

export type OptionsExpiryDetailProps = {
  expiry?: MarketOptionsExpiry;
  indexPrice?: string;
  isLoading?: boolean;
};
