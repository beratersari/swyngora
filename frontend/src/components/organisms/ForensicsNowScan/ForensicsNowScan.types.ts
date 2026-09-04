import type { ForensicsScan } from '@/libs/types/forensics.types';

export type ForensicsNowScanProps = {
  scan?: ForensicsScan;
  isLoading?: boolean;
  onOpenSymbol?: (symbol: string) => void;
};
