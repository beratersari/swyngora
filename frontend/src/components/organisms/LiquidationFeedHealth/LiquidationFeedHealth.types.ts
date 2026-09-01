import type { MarketLiquidationOverview, MarketLiquidations } from '@/libs/api';

export type LiquidationFeedHealthProps = {
  feed?: MarketLiquidations['feed'] | MarketLiquidationOverview['feed'] | null;
};