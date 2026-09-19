import type { WithLoadingProps } from '@/components/types';
import type {
  MarketCvd,
  MarketLiquidations,
  MarketOpenInterest,
  MarketPrintSize,
  MarketSnapshot,
} from '@/libs/api';

export type TapePanelProps = WithLoadingProps & {
  openInterest?: MarketOpenInterest | null;
  openInterestError?: string | null;
  liquidations?: MarketLiquidations | null;
  liquidationsError?: string | null;
  cvd?: MarketCvd | null;
  cvdError?: string | null;
  snapshot?: MarketSnapshot | null;
  snapshotError?: string | null;
  printSize?: MarketPrintSize | null;
  printSizeError?: string | null;
};
