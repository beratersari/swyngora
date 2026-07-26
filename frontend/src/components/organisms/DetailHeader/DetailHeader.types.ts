import type { WithLoadingProps } from '@/components/types';

export type DetailHeaderProps = WithLoadingProps & {
  symbol: string;
  exchange: string;
  lastPrice?: string | number | null;
  priceChangePercent?: string | number | null;
  assetName?: string | null;
  backTo?: string;
};
