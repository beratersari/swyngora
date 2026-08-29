import type { AssetHolders } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type HolderWalletRow = NonNullable<NonNullable<AssetHolders['topHolders']>[number]>;

export type HolderPanelProps = WithLoadingProps & {
  holders?: AssetHolders | null;
  error?: string | null;
};
