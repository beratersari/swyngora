import type { LiqHeatRange, LiqHeatVenue } from '@/components/organisms/LiquidationHeatmap';
import type { LiquidationOpenLevels } from '@/libs/api';

export type OpenLiquidationLevelsProps = {
  data?: LiquidationOpenLevels;
  venue: LiqHeatVenue;
  range: LiqHeatRange;
  onRangeChange?: (range: LiqHeatRange) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
};
