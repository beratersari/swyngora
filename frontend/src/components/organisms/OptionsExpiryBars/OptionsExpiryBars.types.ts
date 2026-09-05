import type { MarketOptionsExpiry } from '@/libs/api';

export type OptionsExpiryBarsProps = {
  expiries?: MarketOptionsExpiry[];
  selectedKey?: string;
  onSelect?: (key: string) => void;
  isLoading?: boolean;
};
