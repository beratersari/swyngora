import type { WithLoadingProps } from '@/components/types';

export type DetailHeaderProps = WithLoadingProps & {
  symbol: string;
  exchange: string;
  lastPrice?: string | number | null;
  priceChangePercent?: string | number | null;
  assetName?: string | null;
  logoUrl?: string | null;
  listingDate?: string | null;
  contractLabel?: string | null;
  backTo?: string;
  watched?: boolean;
  onToggleWatch?: () => void;
  watchLoading?: boolean;
  alertTo?: string;
  compareTo?: string;
  signalsTo?: string;
  delistTime?: string | null;
  announcedAt?: string | null;
  halted?: boolean;
};
