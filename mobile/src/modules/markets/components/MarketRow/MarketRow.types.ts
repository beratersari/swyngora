import type { MarketRowViewModel } from '../../markets.types';

export type MarketRowProps = {
  row: MarketRowViewModel;
  onPress?: (symbol: string) => void;
};
