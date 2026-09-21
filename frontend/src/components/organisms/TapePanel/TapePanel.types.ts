import type { WithLoadingProps } from '@/components/types';
import type {
  MarketCvd,
  MarketLiquidations,
  MarketOpenInterest,
  MarketParticipation,
  MarketVolumeShare,
  MarketPrintSize,
  MarketPressure,
  MarketSnapshot,
  MarketPerpDepthOverview,
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
  participation?: MarketParticipation | null;
  participationError?: string | null;
  volumeShare?: MarketVolumeShare | null;
  volumeShareError?: string | null;
  perpLiquidity?: MarketPerpDepthOverview | null;
  perpLiquidityError?: string | null;
  symbol?: string | null;
  printSize?: MarketPrintSize | null;
  printSizeError?: string | null;
  pressure?: MarketPressure | null;
  pressureError?: string | null;
};
