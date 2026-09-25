import type { MarketOptionsStrikeOIWindow } from '@/libs/api';

export type OptionsStrikeOIProps = {
  windows?: MarketOptionsStrikeOIWindow[];
  expiry?: string;
  isLoading?: boolean;
};
