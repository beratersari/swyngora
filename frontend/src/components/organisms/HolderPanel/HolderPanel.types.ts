import type { AssetHolders } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type HolderPanelProps = WithLoadingProps & {
  holders?: AssetHolders | null;
  error?: string | null;
  /** Circulating supply of the base asset — used to scale dust-sized CMC balances. */
  circulatingSupply?: number | null;
  /** USD price for an estimated wallet value column. */
  priceUsd?: number | null;
};
