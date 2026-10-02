import type { MarketUnusualScanHit } from '@/libs/api';

export type UnusualScanBoardProps = {
  hits: MarketUnusualScanHit[];
  loading?: boolean;
};
